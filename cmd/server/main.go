package main

import (
	"github.com/JorgeSaicoski/microservice-commons/config"
	"github.com/JorgeSaicoski/microservice-commons/database"
	"github.com/JorgeSaicoski/microservice-commons/server"
	"github.com/JorgeSaicoski/microservice-commons/utils"
	"github.com/JorgeSaicoski/professional-tracker/internal/api/projects"
	"github.com/JorgeSaicoski/professional-tracker/internal/api/sessions"
	clients "github.com/JorgeSaicoski/professional-tracker/internal/client"
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

	coreURL := utils.GetEnv("PROJECT_CORE_URL", "http://localhost:8000/api/internal")

	coreClient := clients.NewCoreProjectHTTPClient(coreURL)

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
	projectService := projectsService.NewProfessionalProjectService(dbConnection, coreClient)
	sessionService := sessionsService.NewTimeSessionService(dbConnection)

	// Setup routes
	api := router.Group("")
	projects.RegisterRoutes(api, projectService)
	sessions.RegisterRoutes(api, sessionService)
}
