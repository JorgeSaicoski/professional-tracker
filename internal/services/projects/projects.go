package projects

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/JorgeSaicoski/pgconnect"

	"log/slog"

	clients "github.com/JorgeSaicoski/professional-tracker/internal/client"
	"github.com/JorgeSaicoski/professional-tracker/internal/db"
)

/* ------------------------------------------------------------------ */
/*  Logger                                                            */
/* ------------------------------------------------------------------ */

var log = slog.Default().With(
	slog.String("layer", "service"),
	slog.String("service", "ProfessionalProjectService"),
)

/* ------------------------------------------------------------------ */
/*  Service definition & constructor                                  */
/* ------------------------------------------------------------------ */

type ProfessionalProjectService struct {
	projectRepo   *pgconnect.Repository[db.ProfessionalProject]
	freelanceRepo *pgconnect.Repository[db.ProjectAssignment]
	sessionRepo   *pgconnect.Repository[db.TimeSession]

	coreClient clients.CoreProjectClient
}

func NewProfessionalProjectService(
	database *pgconnect.DB,
	coreClient clients.CoreProjectClient,
) *ProfessionalProjectService {
	return &ProfessionalProjectService{
		projectRepo:   pgconnect.NewRepository[db.ProfessionalProject](database),
		freelanceRepo: pgconnect.NewRepository[db.ProjectAssignment](database),
		sessionRepo:   pgconnect.NewRepository[db.TimeSession](database),
		coreClient:    coreClient,
	}
}

/* ------------------------------------------------------------------ */
/*  DTOs                                                              */
/* ------------------------------------------------------------------ */

type CreateProfessionalProjectInput struct {
	Title      string  `json:"title"`
	ClientName *string `json:"clientName,omitempty"`
}

/* ------------------------------------------------------------------ */
/*  CRUD – Professional Project                                       */
/* ------------------------------------------------------------------ */

func (s *ProfessionalProjectService) CreateProfessionalProject(
	in *CreateProfessionalProjectInput,
	userID string,
) (*db.ProfessionalProject, error) {
	log.Info("create-professional-project:start", "userID", userID, "title", in.Title)

	bpReq := &clients.BaseProjectCreateRequest{
		Title:   in.Title,
		OwnerID: userID,
		Status:  "active",
	}
	base, err := s.coreClient.CreateBaseProject(context.Background(), bpReq)
	if err != nil {
		log.Error("create-professional-project:core-failed", "err", err)
		return nil, fmt.Errorf("create base project: %w", err)
	}
	if base.ID == "" || base.ID == "0" {
		log.Error("core project ID missing", "got", base.ID)
		return nil, fmt.Errorf("core project ID missing (got %q)", base.ID)
	}

	now := time.Now()
	pp := &db.ProfessionalProject{
		BaseProjectID:   base.ID,
		ClientName:      in.ClientName,
		Title:           in.Title,
		TotalHours:      0,
		TotalSalaryCost: 0,
		IsActive:        true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.projectRepo.Create(pp); err != nil {
		log.Error("create-professional-project:db-insert-failed", "err", err)
		return nil, fmt.Errorf("failed to create professional project: %w", err)
	}

	log.Info("create-professional-project:success", "projectID", pp.ID)
	return pp, nil
}

func (s *ProfessionalProjectService) GetProfessionalProject(
	id uint,
	userID string,
) (*db.ProfessionalProject, error) {
	log.Debug("get-professional-project", "projectID", id, "userID", userID)

	var project db.ProfessionalProject
	if err := s.projectRepo.FindByID(id, &project); err != nil {
		log.Error("get-professional-project:not-found", "err", err)
		return nil, fmt.Errorf("professional project not found: %w", err)
	}

	if err := s.loadProjectRelations(&project); err != nil {
		log.Error("get-professional-project:load-relations-failed", "err", err)
		return nil, fmt.Errorf("failed to load project relations: %w", err)
	}
	return &project, nil
}

func (s *ProfessionalProjectService) UpdateProfessionalProject(
	id uint,
	updates *db.ProfessionalProject,
	userID string,
) (*db.ProfessionalProject, error) {
	log.Info("update-professional-project:start", "projectID", id, "userID", userID)

	var project db.ProfessionalProject
	if err := s.projectRepo.FindByID(id, &project); err != nil {
		log.Error("update-professional-project:not-found", "err", err)
		return nil, fmt.Errorf("professional project not found: %w", err)
	}

	if updates.ClientName != nil {
		project.ClientName = updates.ClientName
	}

	if updates.IsActive != project.IsActive {
		project.IsActive = updates.IsActive
	}
	project.UpdatedAt = time.Now()

	if err := s.projectRepo.Update(&project); err != nil {
		log.Error("update-professional-project:db-update-failed", "err", err)
		return nil, fmt.Errorf("failed to update professional project: %w", err)
	}

	log.Info("update-professional-project:success", "projectID", id)
	return &project, nil
}

func (s *ProfessionalProjectService) DeleteProfessionalProject(
	id uint,
	userID string,
) error {
	log.Info("delete-professional-project:start", "projectID", id, "userID", userID)

	var project db.ProfessionalProject
	if err := s.projectRepo.FindByID(id, &project); err != nil {
		log.Error("delete-professional-project:not-found", "err", err)
		return fmt.Errorf("professional project not found: %w", err)
	}

	var activeSessions []db.TimeSession
	if err := s.sessionRepo.FindWhere(&activeSessions,
		"project_id = ? AND is_active = ?", id, true); err != nil {
		log.Error("delete-professional-project:session-check-failed", "err", err)
		return fmt.Errorf("failed to check active sessions: %w", err)
	}
	if len(activeSessions) > 0 {
		log.Warn("delete-professional-project:active-sessions", "count", len(activeSessions))
		return errors.New("cannot delete project with active time sessions")
	}

	if err := s.projectRepo.Delete(&project); err != nil {
		log.Error("delete-professional-project:db-delete-failed", "err", err)
		return fmt.Errorf("failed to delete professional project: %w", err)
	}

	log.Info("delete-professional-project:success", "projectID", id)
	return nil
}

func (s *ProfessionalProjectService) GetUserProfessionalProjects(
	userID string,
) ([]db.ProfessionalProject, error) {
	log.Debug("list-professional-projects", "userID", userID)

	var projects []db.ProfessionalProject
	if err := s.projectRepo.FindAll(&projects); err != nil {
		log.Error("list-professional-projects:query-failed", "err", err)
		return nil, fmt.Errorf("failed to retrieve professional projects: %w", err)
	}
	for i := range projects {
		if err := s.loadProjectRelations(&projects[i]); err != nil {
			log.Error("list-professional-projects:relation-load-failed",
				"projectID", projects[i].ID, "err", err)
			return nil, fmt.Errorf(
				"failed to load relations for project %d: %w",
				projects[i].ID, err,
			)
		}
	}
	log.Info("list-professional-projects:success", "count", len(projects))
	return projects, nil
}

/* ------------------------------------------------------------------ */
/*  CRUD – Freelance sub-project                                      */
/* ------------------------------------------------------------------ */

func (s *ProfessionalProjectService) CreateProjectAssignment(
	parentProjectID uint,
	freelance *db.ProjectAssignment,
	userID string,
) (*db.ProjectAssignment, error) {
	log.Info("create-freelance-project:start", "parentID", parentProjectID, "userID", userID)

	parentProject, err := s.GetProfessionalProject(parentProjectID, userID)
	if err != nil {
		log.Error("create-freelance-project:parent-invalid", "err", err)
		return nil, fmt.Errorf("invalid parent project: %w", err)
	}

	freelance.ParentProjectID = parentProject.ID
	freelance.IsActive = true
	freelance.HoursDedicated = 0
	freelance.TotalCost = 0
	freelance.CreatedAt = time.Now()
	freelance.UpdatedAt = time.Now()

	if err := s.freelanceRepo.Create(freelance); err != nil {
		log.Error("create-freelance-project:db-insert-failed", "err", err)
		return nil, fmt.Errorf("failed to create freelance project: %w", err)
	}

	log.Info("create-freelance-project:success", "freelanceID", freelance.ID)
	return freelance, nil
}

func (s *ProfessionalProjectService) GetProjectAssignment(
	id uint,
	userID string,
) (*db.ProjectAssignment, error) {
	log.Debug("get-freelance-project", "freelanceID", id, "userID", userID)

	var freelance db.ProjectAssignment
	if err := s.freelanceRepo.FindByID(id, &freelance); err != nil {
		log.Error("get-freelance-project:not-found", "err", err)
		return nil, fmt.Errorf("freelance project not found: %w", err)
	}

	if freelance.WorkerUserID != userID {
		log.Warn("get-freelance-project:access-denied", "freelanceID", id, "userID", userID)
		return nil, errors.New("access denied: freelance project is private to the worker")
	}
	return &freelance, nil
}

func (s *ProfessionalProjectService) UpdateProjectAssignment(
	id uint,
	updates *db.ProjectAssignment,
	userID string,
) (*db.ProjectAssignment, error) {
	log.Info("update-freelance-project:start", "freelanceID", id, "userID", userID)

	freelance, err := s.GetProjectAssignment(id, userID)
	if err != nil {
		return nil, err
	}

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
		log.Error("update-freelance-project:db-update-failed", "err", err)
		return nil, fmt.Errorf("failed to update freelance project: %w", err)
	}

	log.Info("update-freelance-project:success", "freelanceID", id)
	return freelance, nil
}

/* ------------------------------------------------------------------ */
/*  Reporting / Business logic                                        */
/* ------------------------------------------------------------------ */

func (s *ProfessionalProjectService) CalculateProjectTotals(
	projectID uint,
) error {
	log.Info("calc-totals:start", "projectID", projectID)

	var project db.ProfessionalProject
	if err := s.projectRepo.FindByID(projectID, &project); err != nil {
		log.Error("calc-totals:project-not-found", "err", err)
		return err
	}

	var sessions []db.TimeSession
	if err := s.sessionRepo.FindWhere(&sessions,
		"project_id = ? AND session_type = ?", projectID, db.SessionTypeWork); err != nil {
		log.Error("calc-totals:sessions-query-failed", "err", err)
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

	project.TotalHours = totalHours
	project.TotalSalaryCost = totalCost
	project.UpdatedAt = time.Now()

	if err := s.projectRepo.Update(&project); err != nil {
		log.Error("calc-totals:project-update-failed", "err", err)
		return err
	}

	log.Info("calc-totals:success", "projectID", projectID,
		"totalHours", totalHours, "totalCost", totalCost)
	return nil
}

func (s *ProfessionalProjectService) GetProjectCostReport(
	projectID uint,
	userID string,
) (*db.ProjectTimeReport, error) {
	log.Debug("get-cost-report", "projectID", projectID, "userID", userID)

	project, err := s.GetProfessionalProject(projectID, userID)
	if err != nil {
		return nil, err
	}

	var sessions []db.TimeSession
	if err := s.sessionRepo.FindWhere(&sessions, "project_id = ?", projectID); err != nil {
		return nil, err
	}

	report := &db.ProjectTimeReport{
		ProjectID:    projectID,
		ProjectTitle: fmt.Sprintf("Professional Project %d", projectID),
		TotalHours:   project.TotalHours,
		TotalCost:    project.TotalSalaryCost,
		WorkSessions: len(sessions),
	}

	if len(sessions) > 0 {
		report.AverageSession = project.TotalHours / float64(len(sessions))
		for _, session := range sessions {
			if session.CreatedAt.After(report.LastActivity) {
				report.LastActivity = session.CreatedAt
			}
		}
	}

	log.Info("get-cost-report:success", "projectID", projectID, "totalHours", report.TotalHours)
	return report, nil
}

/* ------------------------------------------------------------------ */
/*  Helpers                                                           */
/* ------------------------------------------------------------------ */

func (s *ProfessionalProjectService) loadProjectRelations(
	project *db.ProfessionalProject,
) error {
	if err := s.freelanceRepo.FindWhere(&project.ProjectAssignments,
		"parent_project_id = ?", project.ID); err != nil {
		return fmt.Errorf("failed to load freelance projects: %w", err)
	}

	if err := s.sessionRepo.FindWhere(&project.TimeSessions,
		"project_id = ?", project.ID); err != nil {
		return fmt.Errorf("failed to load time sessions: %w", err)
	}
	return nil
}

func (s *ProfessionalProjectService) calculateSessionDuration(
	session *db.TimeSession,
) int {
	if session.EndTime == nil {
		return int(time.Since(session.StartTime).Minutes())
	}
	return int(session.EndTime.Sub(session.StartTime).Minutes())
}
