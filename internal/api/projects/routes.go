package projects

import (
	"github.com/JorgeSaicoski/professional-tracker/internal/api"
	"github.com/JorgeSaicoski/professional-tracker/internal/services/projects"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers all professional project related routes
func RegisterRoutes(router *gin.RouterGroup, projectService *projects.ProfessionalProjectService) {
	handler := NewProjectHandler(projectService)

	// Professional projects endpoints
	projectsGroup := router.Group("/projects")
	projectsGroup.Use(
		api.LoggingMiddleware(),
		api.AuthMiddleware(),
	)

	{
		// Project CRUD
		projectsGroup.POST("", handler.CreateProfessionalProject)       // Create professional project
		projectsGroup.GET("/:id", handler.GetProfessionalProject)       // Get project by ID
		projectsGroup.PUT("/:id", handler.UpdateProfessionalProject)    // Update project
		projectsGroup.DELETE("/:id", handler.DeleteProfessionalProject) // Delete project

		// User projects
		projectsGroup.GET("", handler.GetUserProfessionalProjects) // Get user's professional projects

		// Freelance sub-projects
		projectsGroup.POST("/:id/freelance", handler.CreateFreelanceProject)             // Create freelance sub-project
		projectsGroup.GET("/:id/freelance/:freelanceId", handler.GetFreelanceProject)    // Get freelance project
		projectsGroup.PUT("/:id/freelance/:freelanceId", handler.UpdateFreelanceProject) // Update freelance project

		// Reports
		projectsGroup.GET("/:id/report", handler.GetProjectCostReport) // Get project cost report
	}
}
