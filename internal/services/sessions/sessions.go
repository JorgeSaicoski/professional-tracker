package sessions

import (
	"errors"
	"fmt"
	"time"

	"github.com/JorgeSaicoski/pgconnect"
	"github.com/JorgeSaicoski/professional-tracker/internal/db"
)

type TimeSessionService struct {
	sessionRepo       *pgconnect.Repository[db.TimeSession]
	breakRepo         *pgconnect.Repository[db.SessionBreak]
	activeSessionRepo *pgconnect.Repository[db.UserActiveSession]
	projectRepo       *pgconnect.Repository[db.ProfessionalProject]
}

func NewTimeSessionService(database *pgconnect.DB) *TimeSessionService {
	return &TimeSessionService{
		sessionRepo:       pgconnect.NewRepository[db.TimeSession](database),
		breakRepo:         pgconnect.NewRepository[db.SessionBreak](database),
		activeSessionRepo: pgconnect.NewRepository[db.UserActiveSession](database),
		projectRepo:       pgconnect.NewRepository[db.ProfessionalProject](database),
	}
}

// StartWorkSession starts a new work session
func (s *TimeSessionService) StartWorkSession(projectID uint, companyID, userID string, hourlyRate *float64) (*db.TimeSession, error) {
	// Check if user already has an active session
	if hasActive, err := s.HasActiveSession(userID); err != nil {
		return nil, fmt.Errorf("failed to check active session: %w", err)
	} else if hasActive {
		return nil, errors.New("user already has an active session - finish current session first")
	}

	// Validate project exists
	var project db.ProfessionalProject
	if err := s.projectRepo.FindByID(projectID, &project); err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	// TODO: Validate user has access to this project via project-core

	// Create new session
	session := &db.TimeSession{
		ProjectID:   projectID,
		UserID:      userID,
		CompanyID:   companyID,
		StartTime:   time.Now(),
		SessionType: db.SessionTypeWork,
		HourlyRate:  hourlyRate,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.sessionRepo.Create(session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Create active session record
	activeSession := &db.UserActiveSession{
		UserID:         userID,
		SessionID:      session.ID,
		CompanyID:      companyID,
		ProjectID:      projectID,
		StartedAt:      session.StartTime,
		LastActivityAt: session.StartTime,
		IsOnBreak:      false,
		UpdatedAt:      time.Now(),
	}

	if err := s.activeSessionRepo.Create(activeSession); err != nil {
		// Rollback session creation
		s.sessionRepo.Delete(session)
		return nil, fmt.Errorf("failed to create active session record: %w", err)
	}

	return session, nil
}

// FinishWorkSession ends the current active session
func (s *TimeSessionService) FinishWorkSession(userID string) (*db.TimeSession, error) {
	// Get active session
	activeSession, err := s.GetActiveSession(userID)
	if err != nil {
		return nil, fmt.Errorf("no active session found: %w", err)
	}

	// End any active break first
	if activeSession.IsOnBreak && activeSession.CurrentBreakID != nil {
		if _, err := s.EndBreak(userID); err != nil {
			return nil, fmt.Errorf("failed to end active break: %w", err)
		}
	}

	// Get the session
	var session db.TimeSession
	if err := s.sessionRepo.FindByID(activeSession.SessionID, &session); err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	// End the session
	now := time.Now()
	session.EndTime = &now
	session.IsActive = false
	session.DurationMinutes = s.calculateSessionDuration(&session)
	session.SessionCost = s.calculateSessionCost(&session)
	session.UpdatedAt = now

	if err := s.sessionRepo.Update(&session); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	// Remove active session record
	if err := s.activeSessionRepo.Delete(activeSession); err != nil {
		return nil, fmt.Errorf("failed to remove active session record: %w", err)
	}

	return &session, nil
}

// TakeBreak starts a break during the current session
func (s *TimeSessionService) TakeBreak(userID, breakType string) (*db.SessionBreak, error) {
	// Get active session
	activeSession, err := s.GetActiveSession(userID)
	if err != nil {
		return nil, fmt.Errorf("no active session found: %w", err)
	}

	if activeSession.IsOnBreak {
		return nil, errors.New("already on break - end current break first")
	}

	// Validate break type
	if !s.isValidBreakType(breakType) {
		return nil, errors.New("invalid break type - use: break, lunch, or brb")
	}

	// Create break record
	breakRecord := &db.SessionBreak{
		SessionID: activeSession.SessionID,
		BreakType: breakType,
		StartTime: time.Now(),
		IsActive:  true,
		CreatedAt: time.Now(),
	}

	if err := s.breakRepo.Create(breakRecord); err != nil {
		return nil, fmt.Errorf("failed to create break record: %w", err)
	}

	// Update active session
	activeSession.IsOnBreak = true
	activeSession.CurrentBreakID = &breakRecord.ID
	activeSession.LastActivityAt = time.Now()
	activeSession.UpdatedAt = time.Now()

	if err := s.activeSessionRepo.Update(activeSession); err != nil {
		return nil, fmt.Errorf("failed to update active session: %w", err)
	}

	return breakRecord, nil
}

// EndBreak ends the current break and resumes work
func (s *TimeSessionService) EndBreak(userID string) (*db.SessionBreak, error) {
	// Get active session
	activeSession, err := s.GetActiveSession(userID)
	if err != nil {
		return nil, fmt.Errorf("no active session found: %w", err)
	}

	if !activeSession.IsOnBreak || activeSession.CurrentBreakID == nil {
		return nil, errors.New("not currently on break")
	}

	// Get break record
	var breakRecord db.SessionBreak
	if err := s.breakRepo.FindByID(*activeSession.CurrentBreakID, &breakRecord); err != nil {
		return nil, fmt.Errorf("break record not found: %w", err)
	}

	// End the break
	now := time.Now()
	breakRecord.EndTime = &now
	breakRecord.IsActive = false
	breakRecord.DurationMinutes = s.calculateBreakDuration(&breakRecord)

	if err := s.breakRepo.Update(&breakRecord); err != nil {
		return nil, fmt.Errorf("failed to update break record: %w", err)
	}

	// Update active session
	activeSession.IsOnBreak = false
	activeSession.CurrentBreakID = nil
	activeSession.LastActivityAt = time.Now()
	activeSession.UpdatedAt = time.Now()

	if err := s.activeSessionRepo.Update(activeSession); err != nil {
		return nil, fmt.Errorf("failed to update active session: %w", err)
	}

	return &breakRecord, nil
}

// SwitchProject switches to a different project within the same company
func (s *TimeSessionService) SwitchProject(userID string, newProjectID uint) (*db.TimeSession, error) {
	// Get current active session
	activeSession, err := s.GetActiveSession(userID)
	if err != nil {
		return nil, fmt.Errorf("no active session found: %w", err)
	}

	// End current session
	currentSession, err := s.FinishWorkSession(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to finish current session: %w", err)
	}

	// Start new session with the same company
	newSession, err := s.StartWorkSession(newProjectID, activeSession.CompanyID, userID, currentSession.HourlyRate)
	if err != nil {
		return nil, fmt.Errorf("failed to start new session: %w", err)
	}

	return newSession, nil
}

// SwitchCompany switches to a different company (ends current session, starts new one)
func (s *TimeSessionService) SwitchCompany(userID, newCompanyID string, newProjectID uint, hourlyRate *float64) (*db.TimeSession, error) {
	// End current session if exists
	if hasActive, _ := s.HasActiveSession(userID); hasActive {
		if _, err := s.FinishWorkSession(userID); err != nil {
			return nil, fmt.Errorf("failed to finish current session: %w", err)
		}
	}

	// Start new session with new company
	newSession, err := s.StartWorkSession(newProjectID, newCompanyID, userID, hourlyRate)
	if err != nil {
		return nil, fmt.Errorf("failed to start new session: %w", err)
	}

	return newSession, nil
}

// GetActiveSession gets the user's current active session
func (s *TimeSessionService) GetActiveSession(userID string) (*db.UserActiveSession, error) {
	var sessions []db.UserActiveSession
	if err := s.activeSessionRepo.FindWhere(&sessions, "user_id = ?", userID); err != nil {
		return nil, fmt.Errorf("query active session: %w", err)
	}
	if len(sessions) == 0 {
		return nil, fmt.Errorf("no active session found for user")
	}
	active := sessions[0]
	return &active, nil
}

// HasActiveSession checks if user has an active session
func (s *TimeSessionService) HasActiveSession(userID string) (bool, error) {
	var sessions []db.UserActiveSession
	if err := s.activeSessionRepo.FindWhere(&sessions, "user_id = ?", userID); err != nil {
		return false, fmt.Errorf("query active session: %w", err)
	}
	return len(sessions) > 0, nil
}

// GetUserSessionHistory gets session history for a user
func (s *TimeSessionService) GetUserSessionHistory(userID string, startDate, endDate *time.Time) ([]db.TimeSession, error) {
	var sessions []db.TimeSession

	query := "user_id = ?"
	args := []interface{}{userID}

	if startDate != nil {
		query += " AND start_time >= ?"
		args = append(args, *startDate)
	}

	if endDate != nil {
		query += " AND start_time <= ?"
		args = append(args, *endDate)
	}

	if err := s.sessionRepo.FindWhere(&sessions, query, args...); err != nil {
		return nil, fmt.Errorf("failed to retrieve session history: %w", err)
	}

	return sessions, nil
}

// GetProjectSessions gets all sessions for a specific project
func (s *TimeSessionService) GetProjectSessions(projectID uint) ([]db.TimeSession, error) {
	var sessions []db.TimeSession
	if err := s.sessionRepo.FindWhere(&sessions, "project_id = ?", projectID); err != nil {
		return nil, fmt.Errorf("failed to retrieve project sessions: %w", err)
	}
	return sessions, nil
}

// GenerateUserTimeReport generates a time report for a user
func (s *TimeSessionService) GenerateUserTimeReport(userID string, projectID uint, startDate, endDate time.Time) (*db.UserTimeReport, error) {
	sessions, err := s.GetUserSessionHistory(userID, &startDate, &endDate)
	if err != nil {
		return nil, err
	}

	// Filter by project if specified
	if projectID > 0 {
		var filteredSessions []db.TimeSession
		for _, session := range sessions {
			if session.ProjectID == projectID {
				filteredSessions = append(filteredSessions, session)
			}
		}
		sessions = filteredSessions
	}

	report := &db.UserTimeReport{
		UserID:       userID,
		ProjectID:    projectID,
		TotalHours:   0,
		WorkSessions: len(sessions),
		BreakMinutes: 0,
	}

	for _, session := range sessions {
		duration := s.calculateSessionDuration(&session)
		hours := float64(duration) / 60.0

		if session.SessionType == db.SessionTypeWork {
			report.TotalHours += hours
		} else {
			report.BreakMinutes += duration
		}

		if session.CreatedAt.After(report.LastSession) {
			report.LastSession = session.CreatedAt
		}
	}

	report.ProductiveHours = report.TotalHours - (float64(report.BreakMinutes) / 60.0)

	// Calculate average daily hours
	days := endDate.Sub(startDate).Hours() / 24
	if days > 0 {
		report.AverageDaily = report.TotalHours / days
	}

	return report, nil
}

// Helper methods

// calculateSessionDuration calculates session duration in minutes
func (s *TimeSessionService) calculateSessionDuration(session *db.TimeSession) int {
	if session.EndTime == nil {
		return int(time.Since(session.StartTime).Minutes())
	}
	return int(session.EndTime.Sub(session.StartTime).Minutes())
}

// calculateSessionCost calculates session cost
func (s *TimeSessionService) calculateSessionCost(session *db.TimeSession) float64 {
	if session.HourlyRate == nil {
		return 0
	}
	duration := s.calculateSessionDuration(session)
	hours := float64(duration) / 60.0
	return hours * (*session.HourlyRate)
}

// calculateBreakDuration calculates break duration in minutes
func (s *TimeSessionService) calculateBreakDuration(breakRecord *db.SessionBreak) int {
	if breakRecord.EndTime == nil {
		return int(time.Since(breakRecord.StartTime).Minutes())
	}
	return int(breakRecord.EndTime.Sub(breakRecord.StartTime).Minutes())
}

// isValidBreakType validates break type
func (s *TimeSessionService) isValidBreakType(breakType string) bool {
	validTypes := []string{db.BreakTypeShort, db.BreakTypeLunch, db.BreakTypeBRB}
	for _, valid := range validTypes {
		if breakType == valid {
			return true
		}
	}
	return false
}
