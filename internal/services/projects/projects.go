package projects

import (
	"errors"
	"fmt"
	"time"

	"github.com/JorgeSaicoski/pgconnect"
	"github.com/JorgeSaicoski/professional-tracker/internal/db"
)

type ProfessionalProjectService struct {
	projectRepo   *pgconnect.Repository[db.ProfessionalProject]
	freelanceRepo *pgconnect.Repository[db.FreelanceProject]
	sessionRepo   *pgconnect.Repository[db.TimeSession]
	// TODO: Add project-core client for validation
}

func NewProfessionalProjectService(database *pgconnect.DB) *ProfessionalProjectService {
	return &ProfessionalProjectService{
		projectRepo:   pgconnect.NewRepository[db.ProfessionalProject](database),
		freelanceRepo: pgconnect.NewRepository[db.FreelanceProject](database),
		sessionRepo:   pgconnect.NewRepository[db.TimeSession](database),
	}
}

// CreateProfessionalProject creates a new professional project
func (s *ProfessionalProjectService) CreateProfessionalProject(project *db.ProfessionalProject, userID string) (*db.ProfessionalProject, error) {
	// TODO: Validate baseProjectId exists in project-core
	// TODO: Check user has access to base project

	// Set defaults
	now := time.Now()
	project.CreatedAt = now
	project.UpdatedAt = now
	project.IsActive = true
	project.TotalHours = 0
	project.TotalSalaryCost = 0

	if err := s.projectRepo.Create(project); err != nil {
		return nil, fmt.Errorf("failed to create professional project: %w", err)
	}

	return project, nil
}

// GetProfessionalProject retrieves a professional project by ID
func (s *ProfessionalProjectService) GetProfessionalProject(id uint, userID string) (*db.ProfessionalProject, error) {
	var project db.ProfessionalProject

	if err := s.projectRepo.FindByID(id, &project); err != nil {
		return nil, fmt.Errorf("professional project not found: %w", err)
	}

	// TODO: Validate user has access to this project via project-core

	// Load relations
	if err := s.loadProjectRelations(&project); err != nil {
		return nil, fmt.Errorf("failed to load project relations: %w", err)
	}

	return &project, nil
}

// UpdateProfessionalProject updates a professional project
func (s *ProfessionalProjectService) UpdateProfessionalProject(id uint, updates *db.ProfessionalProject, userID string) (*db.ProfessionalProject, error) {
	var project db.ProfessionalProject

	if err := s.projectRepo.FindByID(id, &project); err != nil {
		return nil, fmt.Errorf("professional project not found: %w", err)
	}

	// TODO: Validate user can update this project via project-core

	// Update allowed fields
	if updates.ClientName != nil {
		project.ClientName = updates.ClientName
	}
	if updates.SalaryPerHour != nil {
		project.SalaryPerHour = updates.SalaryPerHour
	}
	if updates.IsActive != project.IsActive {
		project.IsActive = updates.IsActive
	}

	project.UpdatedAt = time.Now()

	if err := s.projectRepo.Update(&project); err != nil {
		return nil, fmt.Errorf("failed to update professional project: %w", err)
	}

	return &project, nil
}

// DeleteProfessionalProject deletes a professional project
func (s *ProfessionalProjectService) DeleteProfessionalProject(id uint, userID string) error {
	var project db.ProfessionalProject

	if err := s.projectRepo.FindByID(id, &project); err != nil {
		return fmt.Errorf("professional project not found: %w", err)
	}

	// TODO: Validate user can delete this project via project-core

	// Check for active sessions
	var activeSessions []db.TimeSession
	if err := s.sessionRepo.FindWhere(&activeSessions, "project_id = ? AND is_active = ?", id, true); err != nil {
		return fmt.Errorf("failed to check active sessions: %w", err)
	}

	if len(activeSessions) > 0 {
		return errors.New("cannot delete project with active time sessions")
	}

	if err := s.projectRepo.Delete(&project); err != nil {
		return fmt.Errorf("failed to delete professional project: %w", err)
	}

	return nil
}

// GetUserProfessionalProjects retrieves all professional projects for a user
func (s *ProfessionalProjectService) GetUserProfessionalProjects(userID string) ([]db.ProfessionalProject, error) {
	// TODO: Get user's accessible base projects from project-core
	// TODO: Filter professional projects based on accessible base projects

	var projects []db.ProfessionalProject
	if err := s.projectRepo.FindAll(&projects); err != nil {
		return nil, fmt.Errorf("failed to retrieve professional projects: %w", err)
	}

	// Load relations for each project
	for i := range projects {
		if err := s.loadProjectRelations(&projects[i]); err != nil {
			return nil, fmt.Errorf("failed to load relations for project %d: %w", projects[i].ID, err)
		}
	}

	return projects, nil
}

// CreateFreelanceProject creates a freelance sub-project
func (s *ProfessionalProjectService) CreateFreelanceProject(parentProjectID uint, freelance *db.FreelanceProject, userID string) (*db.FreelanceProject, error) {
	// Validate parent project exists and user has access
	parentProject, err := s.GetProfessionalProject(parentProjectID, userID)
	if err != nil {
		return nil, fmt.Errorf("invalid parent project: %w", err)
	}

	// Set required fields
	freelance.ParentProjectID = parentProject.ID
	freelance.IsActive = true
	freelance.HoursDedicated = 0
	freelance.TotalCost = 0
	freelance.CreatedAt = time.Now()
	freelance.UpdatedAt = time.Now()

	if err := s.freelanceRepo.Create(freelance); err != nil {
		return nil, fmt.Errorf("failed to create freelance project: %w", err)
	}

	return freelance, nil
}

// GetFreelanceProject retrieves a freelance project by ID
func (s *ProfessionalProjectService) GetFreelanceProject(id uint, userID string) (*db.FreelanceProject, error) {
	var freelance db.FreelanceProject

	if err := s.freelanceRepo.FindByID(id, &freelance); err != nil {
		return nil, fmt.Errorf("freelance project not found: %w", err)
	}

	// Privacy check - only the worker can access their freelance project
	if freelance.WorkerUserID != userID {
		return nil, errors.New("access denied: freelance project is private to the worker")
	}

	return &freelance, nil
}

// UpdateFreelanceProject updates a freelance project
func (s *ProfessionalProjectService) UpdateFreelanceProject(id uint, updates *db.FreelanceProject, userID string) (*db.FreelanceProject, error) {
	freelance, err := s.GetFreelanceProject(id, userID)
	if err != nil {
		return nil, err
	}

	// Update allowed fields
	if updates.CostPerHour > 0 {
		freelance.CostPerHour = updates.CostPerHour
	}
	if updates.Description != nil {
		freelance.Description = updates.Description
	}
	if updates.IsActive != freelance.IsActive {
		freelance.IsActive = updates.IsActive
	}

	freelance.UpdatedAt = time.Now()

	if err := s.freelanceRepo.Update(freelance); err != nil {
		return nil, fmt.Errorf("failed to update freelance project: %w", err)
	}

	return freelance, nil
}

// Business Logic Methods

// CalculateProjectTotals recalculates and updates project totals
func (s *ProfessionalProjectService) CalculateProjectTotals(projectID uint) error {
	var project db.ProfessionalProject
	if err := s.projectRepo.FindByID(projectID, &project); err != nil {
		return err
	}

	// Get all work sessions for this project
	var sessions []db.TimeSession
	if err := s.sessionRepo.FindWhere(&sessions, "project_id = ? AND session_type = ?", projectID, db.SessionTypeWork); err != nil {
		return err
	}

	totalHours := 0.0
	totalCost := 0.0

	for _, session := range sessions {
		duration := s.calculateSessionDuration(&session)
		hours := float64(duration) / 60.0
		totalHours += hours

		if session.HourlyRate != nil {
			totalCost += hours * (*session.HourlyRate)
		}
	}

	// Update project totals
	project.TotalHours = totalHours
	project.TotalSalaryCost = totalCost
	project.UpdatedAt = time.Now()

	return s.projectRepo.Update(&project)
}

// GetProjectCostReport generates a cost report for a project
func (s *ProfessionalProjectService) GetProjectCostReport(projectID uint, userID string) (*db.ProjectTimeReport, error) {
	project, err := s.GetProfessionalProject(projectID, userID)
	if err != nil {
		return nil, err
	}

	// TODO: Get project title from project-core

	var sessions []db.TimeSession
	if err := s.sessionRepo.FindWhere(&sessions, "project_id = ?", projectID); err != nil {
		return nil, err
	}

	report := &db.ProjectTimeReport{
		ProjectID:    projectID,
		ProjectTitle: fmt.Sprintf("Professional Project %d", projectID), // TODO: Get real title
		TotalHours:   project.TotalHours,
		TotalCost:    project.TotalSalaryCost,
		WorkSessions: len(sessions),
	}

	if len(sessions) > 0 {
		report.AverageSession = project.TotalHours / float64(len(sessions))
		// Find latest session
		for _, session := range sessions {
			if session.CreatedAt.After(report.LastActivity) {
				report.LastActivity = session.CreatedAt
			}
		}
	}

	return report, nil
}

// Helper methods

// loadProjectRelations loads related data for a project
func (s *ProfessionalProjectService) loadProjectRelations(project *db.ProfessionalProject) error {
	// Load freelance projects
	if err := s.freelanceRepo.FindWhere(&project.FreelanceProjects, "parent_project_id = ?", project.ID); err != nil {
		return fmt.Errorf("failed to load freelance projects: %w", err)
	}

	// Load time sessions
	if err := s.sessionRepo.FindWhere(&project.TimeSessions, "project_id = ?", project.ID); err != nil {
		return fmt.Errorf("failed to load time sessions: %w", err)
	}

	return nil
}

// calculateSessionDuration calculates session duration in minutes
func (s *ProfessionalProjectService) calculateSessionDuration(session *db.TimeSession) int {
	if session.EndTime == nil {
		return int(time.Since(session.StartTime).Minutes())
	}
	return int(session.EndTime.Sub(session.StartTime).Minutes())
}
