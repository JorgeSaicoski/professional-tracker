package sessions

import (
	"strconv"
	"time"

	keycloakauth "github.com/JorgeSaicoski/keycloak-auth"
	"github.com/JorgeSaicoski/microservice-commons/responses"
	"github.com/JorgeSaicoski/professional-tracker/internal/services/sessions"
	"github.com/gin-gonic/gin"
)

type SessionHandler struct {
	sessionService *sessions.TimeSessionService
}

func NewSessionHandler(sessionService *sessions.TimeSessionService) *SessionHandler {
	return &SessionHandler{
		sessionService: sessionService,
	}
}

func (h *SessionHandler) StartWorkSession(c *gin.Context) {
	var req StartWorkSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.BadRequest(c, err.Error())
		return
	}

	userID, exists := keycloakauth.GetUserID(c)
	if !exists {
		responses.Unauthorized(c, "User not authenticated")
		return
	}

	session, err := h.sessionService.StartWorkSession(req.ProjectID, req.CompanyID, userID, req.HourlyRate)
	if err != nil {
		if err.Error() == "user already has an active session - finish current session first" {
			responses.Conflict(c, err.Error())
			return
		}
		responses.InternalError(c, err.Error())
		return
	}

	response := TimeSessionToResponse(session)
	responses.Created(c, "Work session started successfully", response)
}

func (h *SessionHandler) FinishWorkSession(c *gin.Context) {
	userID, exists := keycloakauth.GetUserID(c)
	if !exists {
		responses.Unauthorized(c, "User not authenticated")
		return
	}

	session, err := h.sessionService.FinishWorkSession(userID)
	if err != nil {
		responses.InternalError(c, err.Error())
		return
	}

	response := TimeSessionToResponse(session)
	responses.Success(c, "Work session finished successfully", response)
}

func (h *SessionHandler) GetActiveSession(c *gin.Context) {
	userID, exists := keycloakauth.GetUserID(c)
	if !exists {
		responses.Unauthorized(c, "User not authenticated")
		return
	}

	activeSession, err := h.sessionService.GetActiveSession(userID)
	if err != nil {
		responses.NotFound(c, "No active session found")
		return
	}

	response := ActiveSessionToResponse(activeSession)
	responses.Success(c, "Active session retrieved successfully", response)
}

func (h *SessionHandler) TakeBreak(c *gin.Context) {
	var req TakeBreakRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.BadRequest(c, err.Error())
		return
	}

	userID, exists := keycloakauth.GetUserID(c)
	if !exists {
		responses.Unauthorized(c, "User not authenticated")
		return
	}

	breakRecord, err := h.sessionService.TakeBreak(userID, req.BreakType)
	if err != nil {
		if err.Error() == "already on break - end current break first" {
			responses.Conflict(c, err.Error())
			return
		}
		if err.Error() == "invalid break type - use: break, lunch, or brb" {
			responses.BadRequest(c, err.Error())
			return
		}
		responses.InternalError(c, err.Error())
		return
	}

	response := SessionBreakToResponse(breakRecord)
	responses.Created(c, "Break started successfully", response)
}

func (h *SessionHandler) EndBreak(c *gin.Context) {
	userID, exists := keycloakauth.GetUserID(c)
	if !exists {
		responses.Unauthorized(c, "User not authenticated")
		return
	}

	breakRecord, err := h.sessionService.EndBreak(userID)
	if err != nil {
		if err.Error() == "not currently on break" {
			responses.BadRequest(c, err.Error())
			return
		}
		responses.InternalError(c, err.Error())
		return
	}

	response := SessionBreakToResponse(breakRecord)
	responses.Success(c, "Break ended successfully, work resumed", response)
}

func (h *SessionHandler) SwitchProject(c *gin.Context) {
	var req SwitchProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.BadRequest(c, err.Error())
		return
	}

	userID, exists := keycloakauth.GetUserID(c)
	if !exists {
		responses.Unauthorized(c, "User not authenticated")
		return
	}

	session, err := h.sessionService.SwitchProject(userID, req.NewProjectID)
	if err != nil {
		responses.InternalError(c, err.Error())
		return
	}

	response := TimeSessionToResponse(session)
	responses.Success(c, "Switched to new project successfully", response)
}

func (h *SessionHandler) SwitchCompany(c *gin.Context) {
	var req SwitchCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.BadRequest(c, err.Error())
		return
	}

	userID, exists := keycloakauth.GetUserID(c)
	if !exists {
		responses.Unauthorized(c, "User not authenticated")
		return
	}

	session, err := h.sessionService.SwitchCompany(userID, req.NewCompanyID, req.NewProjectID, req.HourlyRate)
	if err != nil {
		responses.InternalError(c, err.Error())
		return
	}

	response := TimeSessionToResponse(session)
	responses.Success(c, "Switched to new company successfully", response)
}

func (h *SessionHandler) GetUserSessionHistory(c *gin.Context) {
	userID, exists := keycloakauth.GetUserID(c)
	if !exists {
		responses.Unauthorized(c, "User not authenticated")
		return
	}

	// Parse optional date filters
	var startDate, endDate *time.Time

	if startDateStr := c.Query("startDate"); startDateStr != "" {
		if parsed, err := time.Parse("2006-01-02", startDateStr); err == nil {
			startDate = &parsed
		}
	}

	if endDateStr := c.Query("endDate"); endDateStr != "" {
		if parsed, err := time.Parse("2006-01-02", endDateStr); err == nil {
			endDate = &parsed
		}
	}

	sessions, err := h.sessionService.GetUserSessionHistory(userID, startDate, endDate)
	if err != nil {
		responses.InternalError(c, err.Error())
		return
	}

	sessionResponses := TimeSessionsToResponse(sessions)
	responses.Success(c, "Session history retrieved successfully", gin.H{
		"sessions": sessionResponses,
		"total":    len(sessionResponses),
	})
}

func (h *SessionHandler) GetProjectSessions(c *gin.Context) {
	projectIDParam := c.Param("projectId")
	projectID, err := strconv.ParseUint(projectIDParam, 10, 32)
	if err != nil {
		responses.BadRequest(c, "Invalid project ID")
		return
	}

	_, exists := keycloakauth.GetUserID(c)
	if !exists {
		responses.Unauthorized(c, "User not authenticated")
		return
	}

	// TODO: Validate user has access to this project

	sessions, err := h.sessionService.GetProjectSessions(uint(projectID))
	if err != nil {
		responses.InternalError(c, err.Error())
		return
	}

	sessionResponses := TimeSessionsToResponse(sessions)
	responses.Success(c, "Project sessions retrieved successfully", gin.H{
		"sessions": sessionResponses,
		"total":    len(sessionResponses),
	})
}

func (h *SessionHandler) GenerateUserTimeReport(c *gin.Context) {
	userID, exists := keycloakauth.GetUserID(c)
	if !exists {
		responses.Unauthorized(c, "User not authenticated")
		return
	}

	// Parse query parameters
	projectIDParam := c.Query("projectId")
	startDateParam := c.Query("startDate")
	endDateParam := c.Query("endDate")

	var projectID uint
	if projectIDParam != "" {
		if id, err := strconv.ParseUint(projectIDParam, 10, 32); err == nil {
			projectID = uint(id)
		}
	}

	// Parse dates
	startDate, err := time.Parse("2006-01-02", startDateParam)
	if err != nil {
		responses.BadRequest(c, "Invalid start date format. Use YYYY-MM-DD")
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateParam)
	if err != nil {
		responses.BadRequest(c, "Invalid end date format. Use YYYY-MM-DD")
		return
	}

	report, err := h.sessionService.GenerateUserTimeReport(userID, projectID, startDate, endDate)
	if err != nil {
		responses.InternalError(c, err.Error())
		return
	}

	response := UserTimeReportToResponse(report, startDateParam, endDateParam)
	responses.Success(c, "Time report generated successfully", response)
}
