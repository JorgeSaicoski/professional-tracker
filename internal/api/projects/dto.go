package projects

import (
	"time"

	"github.com/JorgeSaicoski/professional-tracker/internal/db"
	svc "github.com/JorgeSaicoski/professional-tracker/internal/services/projects"
)

// Request DTOs

// matches the JSON sent by the front-end
type CreateProfessionalProjectRequest struct {
	Title      string  `json:"title" binding:"required"`
	ClientName *string `json:"clientName,omitempty"`
}

type UpdateProfessionalProjectRequest struct {
	ClientName *string `json:"clientName"`
	IsActive   *bool   `json:"isActive"`
}

type CreateFreelanceProjectRequest struct {
	WorkerUserID string  `json:"workerUserId" binding:"required"`
	CostPerHour  float64 `json:"costPerHour" binding:"required"`
	Description  *string `json:"description"`
}

type UpdateFreelanceProjectRequest struct {
	CostPerHour float64 `json:"costPerHour"`
	Description *string `json:"description"`
	IsActive    *bool   `json:"isActive"`
}

// Response DTOs

type ProfessionalProjectResponse struct {
	ID                uint                       `json:"id"`
	BaseProjectID     string                     `json:"baseProjectId"`
	ClientName        *string                    `json:"clientName"`
	TotalSalaryCost   float64                    `json:"totalSalaryCost"`
	TotalHours        float64                    `json:"totalHours"`
	IsActive          bool                       `json:"isActive"`
	CreatedAt         time.Time                  `json:"createdAt"`
	UpdatedAt         time.Time                  `json:"updatedAt"`
	FreelanceProjects []FreelanceProjectResponse `json:"freelanceProjects,omitempty"`
	TimeSessions      []TimeSessionResponse      `json:"timeSessions,omitempty"`
}

type FreelanceProjectResponse struct {
	ID              uint      `json:"id"`
	ParentProjectID uint      `json:"parentProjectId"`
	WorkerUserID    string    `json:"workerUserId"`
	CostPerHour     float64   `json:"costPerHour"`
	HoursDedicated  float64   `json:"hoursDedicated"`
	TotalCost       float64   `json:"totalCost"`
	Description     *string   `json:"description"`
	IsActive        bool      `json:"isActive"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type TimeSessionResponse struct {
	ID                 uint       `json:"id"`
	ProjectID          uint       `json:"projectId"`
	FreelanceProjectID *uint      `json:"freelanceProjectId"`
	UserID             string     `json:"userId"`
	CompanyID          string     `json:"companyId"`
	StartTime          time.Time  `json:"startTime"`
	EndTime            *time.Time `json:"endTime"`
	SessionType        string     `json:"sessionType"`
	DurationMinutes    int        `json:"durationMinutes"`
	HourlyRate         *float64   `json:"hourlyRate"`
	SessionCost        float64    `json:"sessionCost"`
	Notes              *string    `json:"notes"`
	IsActive           bool       `json:"isActive"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

// Conversion methods

func (r *CreateProfessionalProjectRequest) ToProfessionalProject() *db.ProfessionalProject {
	return &db.ProfessionalProject{
		ClientName: r.ClientName,
	}
}

func (r *UpdateProfessionalProjectRequest) ToProfessionalProject() *db.ProfessionalProject {
	project := &db.ProfessionalProject{
		ClientName: r.ClientName,
	}

	if r.IsActive != nil {
		project.IsActive = *r.IsActive
	}

	return project
}

func (r *CreateFreelanceProjectRequest) ToFreelanceProject() *db.FreelanceProject {
	return &db.FreelanceProject{
		WorkerUserID: r.WorkerUserID,
		CostPerHour:  r.CostPerHour,
		Description:  r.Description,
	}
}

func (r *UpdateFreelanceProjectRequest) ToFreelanceProject() *db.FreelanceProject {
	project := &db.FreelanceProject{
		CostPerHour: r.CostPerHour,
		Description: r.Description,
	}

	if r.IsActive != nil {
		project.IsActive = *r.IsActive
	}

	return project
}

// New helper ➜ turns the API request into the service-layer input
func (r *CreateProfessionalProjectRequest) ToInput() *svc.CreateProfessionalProjectInput {
	return &svc.CreateProfessionalProjectInput{
		Title:      r.Title,
		ClientName: r.ClientName,
	}
}

func ProfessionalProjectToResponse(project *db.ProfessionalProject) ProfessionalProjectResponse {
	response := ProfessionalProjectResponse{
		ID:              project.ID,
		BaseProjectID:   project.BaseProjectID,
		ClientName:      project.ClientName,
		TotalSalaryCost: project.TotalSalaryCost,
		TotalHours:      project.TotalHours,
		IsActive:        project.IsActive,
		CreatedAt:       project.CreatedAt,
		UpdatedAt:       project.UpdatedAt,
	}

	// Convert freelance projects
	if len(project.FreelanceProjects) > 0 {
		response.FreelanceProjects = make([]FreelanceProjectResponse, len(project.FreelanceProjects))
		for i, fp := range project.FreelanceProjects {
			response.FreelanceProjects[i] = FreelanceProjectToResponse(&fp)
		}
	}

	// Convert time sessions
	if len(project.TimeSessions) > 0 {
		response.TimeSessions = make([]TimeSessionResponse, len(project.TimeSessions))
		for i, ts := range project.TimeSessions {
			response.TimeSessions[i] = TimeSessionToResponse(&ts)
		}
	}

	return response
}

func ProfessionalProjectsToResponse(projects []db.ProfessionalProject) []ProfessionalProjectResponse {
	responses := make([]ProfessionalProjectResponse, len(projects))
	for i, project := range projects {
		responses[i] = ProfessionalProjectToResponse(&project)
	}
	return responses
}

func FreelanceProjectToResponse(project *db.FreelanceProject) FreelanceProjectResponse {
	return FreelanceProjectResponse{
		ID:              project.ID,
		ParentProjectID: project.ParentProjectID,
		WorkerUserID:    project.WorkerUserID,
		CostPerHour:     project.CostPerHour,
		HoursDedicated:  project.HoursDedicated,
		TotalCost:       project.TotalCost,
		Description:     project.Description,
		IsActive:        project.IsActive,
		CreatedAt:       project.CreatedAt,
		UpdatedAt:       project.UpdatedAt,
	}
}

func TimeSessionToResponse(session *db.TimeSession) TimeSessionResponse {
	return TimeSessionResponse{
		ID:                 session.ID,
		ProjectID:          session.ProjectID,
		FreelanceProjectID: session.FreelanceProjectID,
		UserID:             session.UserID,
		CompanyID:          session.CompanyID,
		StartTime:          session.StartTime,
		EndTime:            session.EndTime,
		SessionType:        session.SessionType,
		DurationMinutes:    session.DurationMinutes,
		HourlyRate:         session.HourlyRate,
		SessionCost:        session.SessionCost,
		Notes:              session.Notes,
		IsActive:           session.IsActive,
		CreatedAt:          session.CreatedAt,
		UpdatedAt:          session.UpdatedAt,
	}
}
