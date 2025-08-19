package sessions

import (
	"errors"
	"log"
	"strconv"
	"time"

	keycloakauth "github.com/JorgeSaicoski/keycloak-auth"
	"github.com/JorgeSaicoski/microservice-commons/responses"
	"github.com/JorgeSaicoski/professional-tracker/internal/services/sessions"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SessionHandler struct {
	sessionService *sessions.TimeSessionService
}

type ActiveSessionEnvelope struct {
	Active  bool                       `json:"active"`
	Session *UserActiveSessionResponse `json:"session"` // nil when no active session
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
	// 1. Log entry point for the handler.
	log.Println("DEBUG: Entering GetActiveSession handler")

	userID, exists := keycloakauth.GetUserID(c)
	if !exists {
		// 2. Log if user ID is not found (authentication failure).
		log.Println("DEBUG: User ID not found in context (unauthenticated)")
		responses.Unauthorized(c, "User not authenticated")
		return
	}
	// 3. Log the retrieved user ID.
	log.Printf("DEBUG: Authenticated userID: %s", userID)

	// service call (adjust name/signature if different in your code)
	active, err := h.sessionService.GetActiveSession(userID)
	// 4. Log after the service call, indicating success or error.
	log.Printf("DEBUG: Call to sessionService.GetActiveSession for userID %s returned (active: %+v, err: %v)", userID, active, err)

	// Not found → 200 with {active:false, session:null}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 5. Log specific case: record not found.
			log.Println("DEBUG: No active session found for the user (gorm.ErrRecordNotFound)")
			responses.Success(c, "ok", ActiveSessionEnvelope{
				Active:  false,
				Session: nil,
			})
			return
		}
		// 6. Log for unexpected errors from the service.
		log.Printf("ERROR: Unexpected error from sessionService.GetActiveSession: %v", err)
		// Unexpected error
		responses.InternalError(c, "failed to get active session")
		return
	}

	// Defensive: service returned nil w/o error → treat as no active session
	if active == nil {
		// 7. Log for a defensive check: service returned nil without an error.
		log.Println("DEBUG: sessionService.GetActiveSession returned nil session without an error (treating as no active session)")
		responses.Success(c, "ok", ActiveSessionEnvelope{
			Active:  false,
			Session: nil,
		})
		return
	}

	// Normal happy path
	// 8. Log the active session object before transforming it.
	log.Printf("DEBUG: Active session found: %+v", active)
	resp := ActiveSessionToResponse(active)
	// 9. Log the transformed response object.
	log.Printf("DEBUG: Transformed session response: %+v", resp)

	responses.Success(c, "ok", ActiveSessionEnvelope{
		Active:  true,
		Session: &resp,
	})
	// 10. Log exit point of the handler.
	log.Println("DEBUG: Exiting GetActiveSession handler (active session returned)")
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
