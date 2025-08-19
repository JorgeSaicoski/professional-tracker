package api

import (
	keycloakauth "github.com/JorgeSaicoski/keycloak-auth"
	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	config := keycloakauth.DefaultConfig()
	config.LoadFromEnv() // Loads KEYCLOAK_URL and KEYCLOAK_REALM

	config.SkipPaths = []string{"/health"}
	config.RequiredClaims = []string{"sub", "preferred_username"}

	tokenAuth := keycloakauth.SimpleAuthMiddleware(config)

	return func(c *gin.Context) {
		// If the upstream gateway already authenticated the user and
		// provided the ID, trust that header and skip JWT validation.
		if userID := c.GetHeader("X-User-ID"); userID != "" {
			c.Set("userID", userID)
			c.Next()
			return
		}

		// Fallback to standard JWT based authentication.
		tokenAuth(c)
	}
}
