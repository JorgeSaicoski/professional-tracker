package sessions

import (
	"github.com/JorgeSaicoski/microservice-commons/middleware"
	"github.com/JorgeSaicoski/professional-tracker/internal/api"
	"github.com/JorgeSaicoski/professional-tracker/internal/services/sessions"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers all time session related routes
func RegisterRoutes(router *gin.RouterGroup, sessionService *sessions.TimeSessionService) {
	handler := NewSessionHandler(sessionService)

	// Time sessions endpoints
	sessionsGroup := router.Group("/sessions")
	sessionsGroup.Use(
		middleware.DefaultLoggingMiddleware(),
		api.AuthMiddleware(),
	)
	{
		// Session management
		sessionsGroup.POST("/start", handler.StartWorkSession)   // Start work session
		sessionsGroup.POST("/finish", handler.FinishWorkSession) // Finish current session
		sessionsGroup.GET("/active", handler.GetActiveSession)   // Get current active session

		// Break management
		sessionsGroup.POST("/break", handler.TakeBreak) // Take a break
		sessionsGroup.POST("/resume", handler.EndBreak) // End break and resume work

		// Project and company switching
		sessionsGroup.POST("/switch-project", handler.SwitchProject) // Switch to different project
		sessionsGroup.POST("/switch-company", handler.SwitchCompany) // Switch to different company

		// Session history and reports
		sessionsGroup.GET("/history", handler.GetUserSessionHistory)         // Get user's session history
		sessionsGroup.GET("/project/:projectId", handler.GetProjectSessions) // Get sessions for a project
		sessionsGroup.GET("/report", handler.GenerateUserTimeReport)         // Generate user time report
	}
}
