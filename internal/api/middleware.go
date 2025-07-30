package api

import (
	keycloakauth "github.com/JorgeSaicoski/keycloak-auth"
	"github.com/JorgeSaicoski/microservice-commons/middleware"
	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	config := keycloakauth.DefaultConfig()
	config.LoadFromEnv() // Loads KEYCLOAK_URL and KEYCLOAK_REALM

	config.SkipPaths = []string{"/health"}
	config.RequiredClaims = []string{"sub", "preferred_username"}

	return keycloakauth.SimpleAuthMiddleware(config)
}

func LoggingMiddleware() gin.HandlerFunc {
	return middleware.DefaultLoggingMiddleware()
}
