package main

import (
	"github.com/JorgeSaicoski/microservice-commons/config"
	"github.com/JorgeSaicoski/microservice-commons/database"
	"github.com/JorgeSaicoski/microservice-commons/server"
	"github.com/JorgeSaicoski/professional-tracker/internal/api/projects"
	"github.com/JorgeSaicoski/professional-tracker/internal/api/sessions"
	"github.com/JorgeSaicoski/professional-tracker/internal/db"
	projectsService "github.com/JorgeSaicoski/professional-tracker/internal/services/projects"
	sessionsService "github.com/JorgeSaicoski/professional-tracker/internal/services/sessions"
	"github.com/gin-gonic/gin"
)

func main() {
	server := server.NewServer(server.ServerOptions{
		ServiceName:    "professional-tracker",
		ServiceVersion: "1.0.0",
		SetupRoutes:    setupRoutes,
	})
	server.Start()
}

func setupRoutes(router *gin.Engine, cfg *config.Config) {
	// Connect to database using microservice-commons
	dbConnection, err := database.ConnectWithConfig(cfg.DatabaseConfig)
	if err != nil {
		panic("Failed to connect to database: " + err.Error())
	}

	// Auto-migrate models
	if err := database.QuickMigrate(dbConnection,
		&db.ProfessionalProject{},
		&db.FreelanceProject{},
		&db.TimeSession{},
		&db.SessionBreak{},
		&db.UserActiveSession{},
	); err != nil {
		panic("Failed to migrate database: " + err.Error())
	}

	// Initialize services
	projectService := projectsService.NewProfessionalProjectService(dbConnection)
	sessionService := sessionsService.NewTimeSessionService(dbConnection)

	// Setup routes
	api := router.Group("/api")
	projects.RegisterRoutes(api, projectService)
	sessions.RegisterRoutes(api, sessionService)

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "professional-tracker",
			"version": "1.0.0",
		})
	})
}
