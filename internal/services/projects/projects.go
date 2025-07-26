package projects

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/JorgeSaicoski/pgconnect"

	clients "github.com/JorgeSaicoski/professional-tracker/internal/client"
	"github.com/JorgeSaicoski/professional-tracker/internal/db"
)

/* ------------------------------------------------------------------ */
/*  Service definition & constructor                                  */
/* ------------------------------------------------------------------ */

// ProfessionalProjectService contains all persistence + integration deps.
type ProfessionalProjectService struct {
	projectRepo   *pgconnect.Repository[db.ProfessionalProject]
	freelanceRepo *pgconnect.Repository[db.FreelanceProject]
	sessionRepo   *pgconnect.Repository[db.TimeSession]

	coreClient clients.CoreProjectClient // ← project-core internal API
}

// NewProfessionalProjectService wires a service with its repositories
// and the CoreProjectClient used for cross-service calls.
func NewProfessionalProjectService(
	database *pgconnect.DB,
	coreClient clients.CoreProjectClient,
) *ProfessionalProjectService {
	return &ProfessionalProjectService{
		projectRepo:   pgconnect.NewRepository[db.ProfessionalProject](database),
		freelanceRepo: pgconnect.NewRepository[db.FreelanceProject](database),
		sessionRepo:   pgconnect.NewRepository[db.TimeSession](database),
		coreClient:    coreClient,
	}
}

/* ------------------------------------------------------------------ */
/*  DTOs                                                              */
/* ------------------------------------------------------------------ */

// CreateProfessionalProjectInput is the payload the handler passes in.
// It mirrors what the front-end sends (title, plus optional extras).
type CreateProfessionalProjectInput struct {
	Title      string  `json:"title"`
	ClientName *string `json:"clientName,omitempty"`
}

/* ------------------------------------------------------------------ */
/*  CRUD – Professional Project                                       */
/* ------------------------------------------------------------------ */

// CreateProfessionalProject creates BOTH the BaseProject (in project-core)
// and the ProfessionalProject (local DB) in one call.
func (s *ProfessionalProjectService) CreateProfessionalProject(
	in *CreateProfessionalProjectInput,
	userID string,
) (*db.ProfessionalProject, error) {

	/* 1️⃣  Create BaseProject through project-core */
	bpReq := &clients.BaseProjectCreateRequest{
		Title:   in.Title,
		OwnerID: userID,
		Status:  "active",
		// CompanyID: nil (empty for now)
	}
	base, err := s.coreClient.CreateBaseProject(context.Background(), bpReq)
	if err != nil {
		return nil, fmt.Errorf("create base project: %w", err)
	}

	/* 2️⃣  Persist ProfessionalProject */
	now := time.Now()
	pp := &db.ProfessionalProject{
		BaseProjectID:   base.ID,
		ClientName:      in.ClientName,
		TotalHours:      0,
		TotalSalaryCost: 0,
		IsActive:        true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.projectRepo.Create(pp); err != nil {
		return nil, fmt.Errorf("failed to create professional project: %w", err)
	}

	return pp, nil
}

// GetProfessionalProject retrieves a professional project by ID
func (s *ProfessionalProjectService) GetProfessionalProject(
	id uint,
	userID string,
) (*db.ProfessionalProject, error) {

	var project db.ProfessionalProject
	if err := s.projectRepo.FindByID(id, &project); err != nil {
		return nil, fmt.Errorf("professional project not found: %w", err)
	}

	// TODO: Validate user access with project-core

	if err := s.loadProjectRelations(&project); err != nil {
		return nil, fmt.Errorf("failed to load project relations: %w", err)
	}
	return &project, nil
}

// UpdateProfessionalProject updates mutable fields
func (s *ProfessionalProjectService) UpdateProfessionalProject(
	id uint,
	updates *db.ProfessionalProject,
	userID string,
) (*db.ProfessionalProject, error) {

	var project db.ProfessionalProject
	if err := s.projectRepo.FindByID(id, &project); err != nil {
		return nil, fmt.Errorf("professional project not found: %w", err)
	}

	// TODO: Validate user access with project-core

	if updates.ClientName != nil {
		project.ClientName = updates.ClientName
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

// DeleteProfessionalProject performs safe deletion (no active sessions)
func (s *ProfessionalProjectService) DeleteProfessionalProject(
	id uint,
	userID string,
) error {

	var project db.ProfessionalProject
	if err := s.projectRepo.FindByID(id, &project); err != nil {
		return fmt.Errorf("professional project not found: %w", err)
	}

	// TODO: Validate user can delete via project-core

	// Prevent deleting while work sessions are active
	var activeSessions []db.TimeSession
	if err := s.sessionRepo.FindWhere(
		&activeSessions,
		"project_id = ? AND is_active = ?",
		id, true,
	); err != nil {
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

// GetUserProfessionalProjects lists all pro-projects visible to the user
func (s *ProfessionalProjectService) GetUserProfessionalProjects(
	userID string,
) ([]db.ProfessionalProject, error) {

	// TODO: Filter by accessible BaseProjects via project-core

	var projects []db.ProfessionalProject
	if err := s.projectRepo.FindAll(&projects); err != nil {
		return nil, fmt.Errorf("failed to retrieve professional projects: %w", err)
	}
	for i := range projects {
		if err := s.loadProjectRelations(&projects[i]); err != nil {
			return nil, fmt.Errorf(
				"failed to load relations for project %d: %w",
				projects[i].ID, err,
			)
		}
	}
	return projects, nil
}

/* ------------------------------------------------------------------ */
/*  CRUD – Freelance sub-project                                      */
/* ------------------------------------------------------------------ */

func (s *ProfessionalProjectService) CreateFreelanceProject(
	parentProjectID uint,
	freelance *db.FreelanceProject,
	userID string,
) (*db.FreelanceProject, error) {

	// Validate parent project
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

func (s *ProfessionalProjectService) GetFreelanceProject(
	id uint,
	userID string,
) (*db.FreelanceProject, error) {

	var freelance db.FreelanceProject
	if err := s.freelanceRepo.FindByID(id, &freelance); err != nil {
		return nil, fmt.Errorf("freelance project not found: %w", err)
	}

	// Only the worker can access their freelance project
	if freelance.WorkerUserID != userID {
		return nil, errors.New("access denied: freelance project is private to the worker")
	}
	return &freelance, nil
}

func (s *ProfessionalProjectService) UpdateFreelanceProject(
	id uint,
	updates *db.FreelanceProject,
	userID string,
) (*db.FreelanceProject, error) {

	freelance, err := s.GetFreelanceProject(id, userID)
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
		return nil, fmt.Errorf("failed to update freelance project: %w", err)
	}
	return freelance, nil
}

/* ------------------------------------------------------------------ */
/*  Reporting / Business logic                                        */
/* ------------------------------------------------------------------ */

// CalculateProjectTotals recomputes hours + cost based on time sessions
func (s *ProfessionalProjectService) CalculateProjectTotals(
	projectID uint,
) error {

	var project db.ProfessionalProject
	if err := s.projectRepo.FindByID(projectID, &project); err != nil {
		return err
	}

	var sessions []db.TimeSession
	if err := s.sessionRepo.FindWhere(
		&sessions,
		"project_id = ? AND session_type = ?",
		projectID, db.SessionTypeWork,
	); err != nil {
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

	return s.projectRepo.Update(&project)
}

// GetProjectCostReport generates a lightweight cost report
func (s *ProfessionalProjectService) GetProjectCostReport(
	projectID uint,
	userID string,
) (*db.ProjectTimeReport, error) {

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
		ProjectTitle: fmt.Sprintf("Professional Project %d", projectID), // TODO fetch real title
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
	return report, nil
}

/* ------------------------------------------------------------------ */
/*  Helpers                                                           */
/* ------------------------------------------------------------------ */

func (s *ProfessionalProjectService) loadProjectRelations(
	project *db.ProfessionalProject,
) error {

	if err := s.freelanceRepo.FindWhere(
		&project.FreelanceProjects,
		"parent_project_id = ?", project.ID,
	); err != nil {
		return fmt.Errorf("failed to load freelance projects: %w", err)
	}

	if err := s.sessionRepo.FindWhere(
		&project.TimeSessions,
		"project_id = ?", project.ID,
	); err != nil {
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
