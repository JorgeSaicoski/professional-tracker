package sessions

import (
	"time"

	"github.com/JorgeSaicoski/professional-tracker/internal/db"
)

// Request DTOs

type StartWorkSessionRequest struct {
	ProjectID  uint     `json:"projectId" binding:"required"`
	CompanyID  string   `json:"companyId" binding:"required"`
	HourlyRate *float64 `json:"hourlyRate"`
}

type TakeBreakRequest struct {
	BreakType string `json:"breakType" binding:"required"` // break, lunch, brb
}

type SwitchProjectRequest struct {
	NewProjectID uint `json:"newProjectId" binding:"required"`
}

type SwitchCompanyRequest struct {
	NewCompanyID string   `json:"newCompanyId" binding:"required"`
	NewProjectID uint     `json:"newProjectId" binding:"required"`
	HourlyRate   *float64 `json:"hourlyRate"`
}

type GenerateReportRequest struct {
	ProjectID uint   `json:"projectId"`
	StartDate string `json:"startDate"` // YYYY-MM-DD format
	EndDate   string `json:"endDate"`   // YYYY-MM-DD format
}

// Response DTOs

type TimeSessionResponse struct {
	ID                  uint       `json:"id"`
	ProjectID           uint       `json:"projectId"`
	ProjectAssignmentID *uint      `json:"projectAssignmentId"`
	UserID              string     `json:"userId"`
	CompanyID           string     `json:"companyId"`
	StartTime           time.Time  `json:"startTime"`
	EndTime             *time.Time `json:"endTime"`
	SessionType         string     `json:"sessionType"`
	DurationMinutes     int        `json:"durationMinutes"`
	HourlyRate          *float64   `json:"hourlyRate"`
	SessionCost         float64    `json:"sessionCost"`
	Notes               *string    `json:"notes"`
	IsActive            bool       `json:"isActive"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
}

type SessionBreakResponse struct {
	ID              uint       `json:"id"`
	SessionID       uint       `json:"sessionId"`
	BreakType       string     `json:"breakType"`
	StartTime       time.Time  `json:"startTime"`
	EndTime         *time.Time `json:"endTime"`
	DurationMinutes int        `json:"durationMinutes"`
	IsActive        bool       `json:"isActive"`
	CreatedAt       time.Time  `json:"createdAt"`
}

type UserActiveSessionResponse struct {
	UserID         string                `json:"userId"`
	SessionID      uint                  `json:"sessionId"`
	CompanyID      string                `json:"companyId"`
	ProjectID      uint                  `json:"projectId"`
	StartedAt      time.Time             `json:"startedAt"`
	LastActivityAt time.Time             `json:"lastActivityAt"`
	IsOnBreak      bool                  `json:"isOnBreak"`
	CurrentBreak   *SessionBreakResponse `json:"currentBreak,omitempty"`
	Session        TimeSessionResponse   `json:"session"`
	UpdatedAt      time.Time             `json:"updatedAt"`
}

type UserTimeReportResponse struct {
	UserID          string    `json:"userId"`
	ProjectID       uint      `json:"projectId"`
	CompanyID       string    `json:"companyId"`
	TotalHours      float64   `json:"totalHours"`
	WorkSessions    int       `json:"workSessions"`
	BreakMinutes    int       `json:"breakMinutes"`
	ProductiveHours float64   `json:"productiveHours"`
	LastSession     time.Time `json:"lastSession"`
	AverageDaily    float64   `json:"averageDaily"`
	StartDate       string    `json:"startDate"`
	EndDate         string    `json:"endDate"`
}

// Conversion methods

func TimeSessionToResponse(session *db.TimeSession) TimeSessionResponse {
	return TimeSessionResponse{
		ID:                  session.ID,
		ProjectID:           session.ProjectID,
		ProjectAssignmentID: session.ProjectAssignmentID,
		UserID:              session.UserID,
		CompanyID:           session.CompanyID,
		StartTime:           session.StartTime,
		EndTime:             session.EndTime,
		SessionType:         session.SessionType,
		DurationMinutes:     session.DurationMinutes,
		HourlyRate:          session.HourlyRate,
		SessionCost:         session.SessionCost,
		Notes:               session.Notes,
		IsActive:            session.IsActive,
		CreatedAt:           session.CreatedAt,
		UpdatedAt:           session.UpdatedAt,
	}
}

func TimeSessionsToResponse(sessions []db.TimeSession) []TimeSessionResponse {
	responses := make([]TimeSessionResponse, len(sessions))
	for i, session := range sessions {
		responses[i] = TimeSessionToResponse(&session)
	}
	return responses
}

func SessionBreakToResponse(breakRecord *db.SessionBreak) SessionBreakResponse {
	return SessionBreakResponse{
		ID:              breakRecord.ID,
		SessionID:       breakRecord.SessionID,
		BreakType:       breakRecord.BreakType,
		StartTime:       breakRecord.StartTime,
		EndTime:         breakRecord.EndTime,
		DurationMinutes: breakRecord.DurationMinutes,
		IsActive:        breakRecord.IsActive,
		CreatedAt:       breakRecord.CreatedAt,
	}
}

func ActiveSessionToResponse(activeSession *db.UserActiveSession) UserActiveSessionResponse {
	response := UserActiveSessionResponse{
		UserID:         activeSession.UserID,
		SessionID:      activeSession.SessionID,
		CompanyID:      activeSession.CompanyID,
		ProjectID:      activeSession.ProjectID,
		StartedAt:      activeSession.StartedAt,
		LastActivityAt: activeSession.LastActivityAt,
		IsOnBreak:      activeSession.IsOnBreak,
		Session:        TimeSessionToResponse(&activeSession.Session),
		UpdatedAt:      activeSession.UpdatedAt,
	}

	if activeSession.CurrentBreak != nil {
		breakResponse := SessionBreakToResponse(activeSession.CurrentBreak)
		response.CurrentBreak = &breakResponse
	}

	return response
}

func UserTimeReportToResponse(report *db.UserTimeReport, startDate, endDate string) UserTimeReportResponse {
	return UserTimeReportResponse{
		UserID:          report.UserID,
		ProjectID:       report.ProjectID,
		CompanyID:       report.CompanyID,
		TotalHours:      report.TotalHours,
		WorkSessions:    report.WorkSessions,
		BreakMinutes:    report.BreakMinutes,
		ProductiveHours: report.ProductiveHours,
		LastSession:     report.LastSession,
		AverageDaily:    report.AverageDaily,
		StartDate:       startDate,
		EndDate:         endDate,
	}
}
