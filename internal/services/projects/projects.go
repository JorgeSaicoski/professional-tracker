package projects

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	projectRepo           *pgconnect.Repository[db.ProfessionalProject]
	projectAssignmentRepo *pgconnect.Repository[db.ProjectAssignment]
	sessionRepo           *pgconnect.Repository[db.TimeSession]

	coreClient clients.CoreProjectClient
}

func NewProfessionalProjectService(
	database *pgconnect.DB,
	coreClient clients.CoreProjectClient,
) *ProfessionalProjectService {
	return &ProfessionalProjectService{
		projectRepo:           pgconnect.NewRepository[db.ProfessionalProject](database),
		projectAssignmentRepo: pgconnect.NewRepository[db.ProjectAssignment](database),
		sessionRepo:           pgconnect.NewRepository[db.TimeSession](database),
		coreClient:            coreClient,
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
	// Backwards-compat wrapper: use request-scoped version with context.Background().
	return s.CreateProfessionalProjectCtx(context.Background(), in, userID)
}

// CreateProfessionalProjectCtx is the request-scoped variant.
func (s *ProfessionalProjectService) CreateProfessionalProjectCtx(
	ctx context.Context,
	in *CreateProfessionalProjectInput,
	userID string,
) (*db.ProfessionalProject, error) {
	bpReq := &clients.BaseProjectCreateRequest{
		Title:   in.Title,
		OwnerID: userID,
		Status:  "active",
	}
	base, err := s.coreClient.CreateBaseProject(ctx, bpReq)
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
	// Backwards-compat wrapper.
	return s.GetProfessionalProjectCtx(context.Background(), id, userID)
}

// GetProfessionalProjectCtx is the request-scoped variant.
func (s *ProfessionalProjectService) GetProfessionalProjectCtx(
	ctx context.Context,
	id uint,
	userID string,
) (*db.ProfessionalProject, error) {

	var project db.ProfessionalProject
	if err := s.projectRepo.FindByID(id, &project); err != nil {
		log.Error("get-professional-project:not-found", "err", err)
		return nil, fmt.Errorf("professional project not found: %w", err)
	}

	// Check access through core service
	_, err := s.coreClient.GetProject(ctx, project.BaseProjectID, userID)
	if err != nil {
		log.Error("get-professional-project:access-denied", "projectID", id, "userID", userID, "err", err)
		return nil, fmt.Errorf("access denied: %w", err)
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
	// Backwards-compat wrapper.
	return s.UpdateProfessionalProjectCtx(context.Background(), id, updates, userID)
}

// UpdateProfessionalProjectCtx is the request-scoped variant.
func (s *ProfessionalProjectService) UpdateProfessionalProjectCtx(
	ctx context.Context,
	id uint,
	updates *db.ProfessionalProject,
	userID string,
) (*db.ProfessionalProject, error) {

	var project db.ProfessionalProject
	if err := s.projectRepo.FindByID(id, &project); err != nil {
		log.Error("update-professional-project:not-found", "err", err)
		return nil, fmt.Errorf("professional project not found: %w", err)
	}

	// NOTE: Permission check using a no-op update (Core requires userId for update authorization).
	// TODO(core-microservice): expose a dedicated "check update permission" endpoint to avoid no-op calls.
	_, err := s.coreClient.UpdateProject(ctx, project.BaseProjectID, userID, &clients.UpdateProjectRequest{})

	if err != nil {
		log.Error("update-professional-project:access-denied", "projectID", id, "userID", userID, "err", err)
		return nil, fmt.Errorf("access denied: %w", err)
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
	// Backwards-compat wrapper.
	return s.DeleteProfessionalProjectCtx(context.Background(), id, userID)
}

// DeleteProfessionalProjectCtx is the request-scoped variant.
func (s *ProfessionalProjectService) DeleteProfessionalProjectCtx(
	ctx context.Context,
	id uint,
	userID string,
) error {
	var project db.ProfessionalProject
	if err := s.projectRepo.FindByID(id, &project); err != nil {
		log.Error("delete-professional-project:not-found", "err", err)
		return fmt.Errorf("professional project not found: %w", err)
	}

	// Check delete permissions through core service (typically owner only)
	err := s.coreClient.DeleteProject(ctx, project.BaseProjectID, userID)
	if err != nil {
		log.Error("delete-professional-project:access-denied", "projectID", id, "userID", userID, "err", err)
		return fmt.Errorf("access denied: %w", err)
	}

	// Check for active sessions before allowing deletion
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
	return s.GetUserProfessionalProjectsPage(context.Background(), userID, 100, 0)
}

// GetUserProfessionalProjectsPage lists projects with pagination (limit/offset) and request-scoped context.
func (s *ProfessionalProjectService) GetUserProfessionalProjectsPage(
	ctx context.Context,
	userID string,
	limit, offset int,
) ([]db.ProfessionalProject, error) {

	// Get base projects that user has access to from core service
	baseProjects, err := s.coreClient.GetUserProjects(ctx, userID)
	if err != nil {
		log.Error("list-professional-projects:core-client-failed", "err", err)
		return nil, fmt.Errorf("failed to get user's base projects: %w", err)
	}

	if len(baseProjects) == 0 {
		log.Info("list-professional-projects:no-base-projects", "userID", userID)
		return []db.ProfessionalProject{}, nil
	}

	// Extract base project IDs for filtering
	baseProjectIDs := make([]interface{}, 0, len(baseProjects))
	for _, bp := range baseProjects {
		baseProjectIDs = append(baseProjectIDs, bp.ID)
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(baseProjectIDs))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	inClause := "base_project_id IN (" + strings.Join(placeholders, ",") + ")"

	// Query professional projects that correspond to user's base projects
	var projects []db.ProfessionalProject
	// NOTE: repository does not support LIMIT/OFFSET directly; if your repo does, apply here.
	// TODO(repo): add Limit/Offset support in pgconnect.Repository; for now, fetch and paginate in-memory.
	if err := s.projectRepo.FindWhere(&projects, inClause, baseProjectIDs...); err != nil {
		log.Error("list-professional-projects:query-failed", "err", err)
		return nil, fmt.Errorf("failed to retrieve professional projects: %w", err)
	}

	// Load relations for the filtered set
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
	// Apply in-memory pagination until repo supports LIMIT/OFFSET.
	start := offset
	if start > len(projects) {
		start = len(projects)
	}
	end := start + limit
	if end > len(projects) {
		end = len(projects)
	}
	paged := projects[start:end]

	// Batch-load relations for the paged set to avoid N+1.
	if err := s.loadRelationsForProjects(paged); err != nil {
		log.Error("list-professional-projects:relation-batch-load-failed", "err", err)
		return nil, fmt.Errorf("failed to load relations: %w", err)
	}

	log.Info("list-professional-projects:success", "count", len(paged), "total", len(projects), "limit", limit, "offset", offset)
	return paged, nil
}

/* ------------------------------------------------------------------ */
/*  CRUD – projectAssignment sub-project                             */
/* ------------------------------------------------------------------ */

func (s *ProfessionalProjectService) CreateProjectAssignment(
	parentProjectID uint,
	projectAssignment *db.ProjectAssignment,
	userID string,
) (*db.ProjectAssignment, error) {
	log.Info("create-projectAssignment-project:start", "parentID", parentProjectID, "userID", userID)
	// Backwards-compat wrapper.
	return s.CreateProjectAssignmentCtx(context.Background(), parentProjectID, projectAssignment, userID)
}

// CreateProjectAssignmentCtx is the request-scoped variant.
func (s *ProfessionalProjectService) CreateProjectAssignmentCtx(
	ctx context.Context,
	parentProjectID uint,
	projectAssignment *db.ProjectAssignment,
	userID string,
) (*db.ProjectAssignment, error) {
	parentProject, err := s.GetProfessionalProjectCtx(ctx, parentProjectID, userID)

	if err != nil {
		log.Error("create-projectAssignment-project:parent-invalid", "err", err)
		return nil, fmt.Errorf("invalid parent project: %w", err)
	}

	projectAssignment.ParentProjectID = parentProject.ID
	projectAssignment.IsActive = true
	projectAssignment.HoursDedicated = 0
	projectAssignment.TotalCost = 0
	projectAssignment.CreatedAt = time.Now()
	projectAssignment.UpdatedAt = time.Now()

	if err := s.projectAssignmentRepo.Create(projectAssignment); err != nil {
		log.Error("create-projectAssignment-project:db-insert-failed", "err", err)
		return nil, fmt.Errorf("failed to create projectAssignment project: %w", err)
	}

	log.Info("create-projectAssignment-project:success", "projectAssignmentID", projectAssignment.ID)
	return projectAssignment, nil
}

func (s *ProfessionalProjectService) GetProjectAssignment(
	id uint,
	userID string,
) (*db.ProjectAssignment, error) {
	log.Debug("get-projectAssignment-project", "projectAssignmentID", id, "userID", userID)
	// Backwards-compat wrapper.
	return s.GetProjectAssignmentCtx(context.Background(), id, userID)
}

// GetProjectAssignmentCtx is the request-scoped variant.
func (s *ProfessionalProjectService) GetProjectAssignmentCtx(
	ctx context.Context,
	id uint,
	userID string,
) (*db.ProjectAssignment, error) {
	var projectAssignment db.ProjectAssignment
	if err := s.projectAssignmentRepo.FindByID(id, &projectAssignment); err != nil {
		log.Error("get-projectAssignment-project:not-found", "err", err)
		return nil, fmt.Errorf("projectAssignment project not found: %w", err)
	}

	// Verify access to parent project
	_, err := s.GetProfessionalProjectCtx(ctx, projectAssignment.ParentProjectID, userID)
	if err != nil {
		log.Warn("get-projectAssignment-project:access-denied", "projectAssignmentID", id, "userID", userID)
		return nil, fmt.Errorf("access denied to parent project: %w", err)
	}

	// Additional check for worker-specific access
	if projectAssignment.WorkerUserID != userID {
		log.Warn("get-projectAssignment-project:worker-access-denied", "projectAssignmentID", id, "userID", userID)
		return nil, errors.New("access denied: projectAssignment project is private to the worker")
	}
	return &projectAssignment, nil
}

func (s *ProfessionalProjectService) UpdateProjectAssignment(
	id uint,
	updates *db.ProjectAssignment,
	userID string,
) (*db.ProjectAssignment, error) {
	// Backwards-compat wrapper.
	return s.UpdateProjectAssignmentCtx(context.Background(), id, updates, userID)
}

// UpdateProjectAssignmentCtx is the request-scoped variant.
func (s *ProfessionalProjectService) UpdateProjectAssignmentCtx(
	ctx context.Context,
	id uint,
	updates *db.ProjectAssignment,
	userID string,
) (*db.ProjectAssignment, error) {
	log.Info("update-projectAssignment-project:start", "projectAssignmentID", id, "userID", userID)

	projectAssignment, err := s.GetProjectAssignmentCtx(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	if updates.CostPerHour > 0 {
		projectAssignment.CostPerHour = updates.CostPerHour
	}
	if updates.Description != nil {
		projectAssignment.Description = updates.Description
	}
	if updates.IsActive != projectAssignment.IsActive {
		projectAssignment.IsActive = updates.IsActive
	}
	projectAssignment.UpdatedAt = time.Now()

	if err := s.projectAssignmentRepo.Update(projectAssignment); err != nil {
		log.Error("update-projectAssignment-project:db-update-failed", "err", err)
		return nil, fmt.Errorf("failed to update projectAssignment project: %w", err)
	}

	log.Info("update-projectAssignment-project:success", "projectAssignmentID", id)
	return projectAssignment, nil
}

func (s *ProfessionalProjectService) GetUserProjectAssignments(
	userID string,
) ([]db.ProjectAssignment, error) {
	log.Debug("get-user-project-assignments", "userID", userID)
	// Backwards-compat wrapper.
	return s.GetUserProjectAssignmentsCtx(context.Background(), userID)
}

// GetUserProjectAssignmentsCtx is the request-scoped variant.
func (s *ProfessionalProjectService) GetUserProjectAssignmentsCtx(
	ctx context.Context,
	userID string,
) ([]db.ProjectAssignment, error) {
	// Get assignments where user is the worker
	var assignments []db.ProjectAssignment
	if err := s.projectAssignmentRepo.FindWhere(&assignments, "worker_user_id = ? AND is_active = ?", userID, true); err != nil {
		log.Error("get-user-project-assignments:query-failed", "err", err)
		return nil, fmt.Errorf("failed to retrieve user project assignments: %w", err)
	}

	log.Info("get-user-project-assignments:success", "userID", userID, "count", len(assignments))
	return assignments, nil
}

func (s *ProfessionalProjectService) GetProjectAssignments(
	projectID uint,
	userID string,
) ([]db.ProjectAssignment, error) {
	log.Debug("get-project-assignments", "projectID", projectID, "userID", userID)
	// Backwards-compat wrapper.
	return s.GetProjectAssignmentsCtx(context.Background(), projectID, userID)
}

// GetProjectAssignmentsCtx is the request-scoped variant.
func (s *ProfessionalProjectService) GetProjectAssignmentsCtx(
	ctx context.Context,
	projectID uint,
	userID string,
) ([]db.ProjectAssignment, error) {
	// Verify user has access to the project
	_, err := s.GetProfessionalProjectCtx(ctx, projectID, userID)
	if err != nil {
		return nil, fmt.Errorf("access denied to project: %w", err)
	}

	var assignments []db.ProjectAssignment
	if err := s.projectAssignmentRepo.FindWhere(&assignments, "parent_project_id = ?", projectID); err != nil {
		log.Error("get-project-assignments:query-failed", "err", err)
		return nil, fmt.Errorf("failed to retrieve project assignments: %w", err)
	}

	log.Info("get-project-assignments:success", "projectID", projectID, "count", len(assignments))
	return assignments, nil
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
	return s.GetProjectCostReportCtx(context.Background(), projectID, userID)
}

// GetProjectCostReportCtx is the request-scoped variant.
func (s *ProfessionalProjectService) GetProjectCostReportCtx(
	ctx context.Context,
	projectID uint,
	userID string,
) (*db.ProjectTimeReport, error) {
	project, err := s.GetProfessionalProjectCtx(ctx, projectID, userID)

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

func (s *ProfessionalProjectService) GetUserTimeReport(
	userID string,
	startDate, endDate *time.Time,
) (*db.UserTimeReport, error) {
	log.Debug("get-user-time-report", "userID", userID)
	return s.GetUserTimeReportCtx(context.Background(), userID, startDate, endDate)
}

// GetUserTimeReportCtx is the request-scoped variant.
func (s *ProfessionalProjectService) GetUserTimeReportCtx(
	ctx context.Context,
	userID string,
	startDate, endDate *time.Time,
) (*db.UserTimeReport, error) {
	// Build query for user's sessions within date range
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

	var sessions []db.TimeSession
	if err := s.sessionRepo.FindWhere(&sessions, query, args...); err != nil {
		log.Error("get-user-time-report:query-failed", "err", err)
		return nil, fmt.Errorf("failed to retrieve user sessions: %w", err)
	}

	report := &db.UserTimeReport{
		UserID:       userID,
		TotalHours:   0,
		WorkSessions: 0,
		BreakMinutes: 0,
	}

	for _, session := range sessions {
		duration := s.calculateSessionDuration(&session)
		hours := float64(duration) / 60.0

		if session.SessionType == db.SessionTypeWork {
			report.TotalHours += hours
			report.WorkSessions++
		} else {
			report.BreakMinutes += duration
		}

		if session.CreatedAt.After(report.LastSession) {
			report.LastSession = session.CreatedAt
		}
	}

	report.ProductiveHours = report.TotalHours - (float64(report.BreakMinutes) / 60.0)

	log.Info("get-user-time-report:success", "userID", userID, "totalHours", report.TotalHours)
	return report, nil
}

/* ------------------------------------------------------------------ */
/*  Helpers                                                           */
/* ------------------------------------------------------------------ */

func (s *ProfessionalProjectService) loadProjectRelations(
	project *db.ProfessionalProject,
) error {
	if err := s.projectAssignmentRepo.FindWhere(&project.ProjectAssignments,
		"parent_project_id = ?", project.ID); err != nil {
		return fmt.Errorf("failed to load projectAssignment projects: %w", err)
	}

	if err := s.sessionRepo.FindWhere(&project.TimeSessions,
		"project_id = ?", project.ID); err != nil {
		return fmt.Errorf("failed to load time sessions: %w", err)
	}
	return nil
}

// loadRelationsForProjects batches relation loading to avoid N+1 queries.
// NOTE: uses IN (...) with placeholders against existing repository API.
func (s *ProfessionalProjectService) loadRelationsForProjects(projects []db.ProfessionalProject) error {
	if len(projects) == 0 {
		return nil
	}
	ids := make([]interface{}, 0, len(projects))
	for _, p := range projects {
		ids = append(ids, p.ID)
	}
	ph := make([]string, len(ids))
	for i := range ph {
		ph[i] = "?"
	}
	in := "(" + strings.Join(ph, ",") + ")"

	// Fetch all assignments for these projects.
	var allAssignments []db.ProjectAssignment
	if err := s.projectAssignmentRepo.FindWhere(&allAssignments, "parent_project_id IN "+in, ids...); err != nil {
		return fmt.Errorf("batch load project assignments: %w", err)
	}
	// Fetch all sessions for these projects.
	var allSessions []db.TimeSession
	if err := s.sessionRepo.FindWhere(&allSessions, "project_id IN "+in, ids...); err != nil {
		return fmt.Errorf("batch load time sessions: %w", err)
	}

	// Group by project id for fast assignment.
	assignmentsByPID := make(map[uint][]db.ProjectAssignment, len(projects))
	for _, a := range allAssignments {
		assignmentsByPID[a.ParentProjectID] = append(assignmentsByPID[a.ParentProjectID], a)
	}
	sessionsByPID := make(map[uint][]db.TimeSession, len(projects))
	for _, s := range allSessions {
		sessionsByPID[s.ProjectID] = append(sessionsByPID[s.ProjectID], s)
	}
	// Attach to the slice elements.
	for i := range projects {
		pid := projects[i].ID
		projects[i].ProjectAssignments = assignmentsByPID[pid]
		projects[i].TimeSessions = sessionsByPID[pid]
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
