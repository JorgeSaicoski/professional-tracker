package projects

import (
	"strconv"

	keycloakauth "github.com/JorgeSaicoski/keycloak-auth"
	"github.com/JorgeSaicoski/microservice-commons/responses"
	"github.com/JorgeSaicoski/professional-tracker/internal/services/projects"
	"github.com/gin-gonic/gin"
)

type ProjectHandler struct {
	projectService *projects.ProfessionalProjectService
}

func NewProjectHandler(projectService *projects.ProfessionalProjectService) *ProjectHandler {
	return &ProjectHandler{
		projectService: projectService,
	}
}

func (h *ProjectHandler) CreateProfessionalProject(c *gin.Context) {
	var req CreateProfessionalProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.BadRequest(c, err.Error())
		return
	}

	userID, exists := keycloakauth.GetUserID(c)
	if !exists {
		responses.Unauthorized(c, "User not authenticated")
		return
	}

	project := req.ToProfessionalProject()
	createdProject, err := h.projectService.CreateProfessionalProject(project, userID)
	if err != nil {
		responses.InternalError(c, err.Error())
		return
	}

	response := ProfessionalProjectToResponse(createdProject)
	responses.Created(c, "Professional project created successfully", response)
}

func (h *ProjectHandler) GetProfessionalProject(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		responses.BadRequest(c, "Invalid project ID")
		return
	}

	userID, exists := keycloakauth.GetUserID(c)
	if !exists {
		responses.Unauthorized(c, "User not authenticated")
		return
	}

	project, err := h.projectService.GetProfessionalProject(uint(id), userID)
	if err != nil {
		responses.NotFound(c, err.Error())
		return
	}

	response := ProfessionalProjectToResponse(project)
	responses.Success(c, "Professional project retrieved successfully", response)
}

func (h *ProjectHandler) UpdateProfessionalProject(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		responses.BadRequest(c, "Invalid project ID")
		return
	}

	var req UpdateProfessionalProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.BadRequest(c, err.Error())
		return
	}

	userID, exists := keycloakauth.GetUserID(c)
	if !exists {
		responses.Unauthorized(c, "User not authenticated")
		return
	}

	updates := req.ToProfessionalProject()
	project, err := h.projectService.UpdateProfessionalProject(uint(id), updates, userID)
	if err != nil {
		responses.InternalError(c, err.Error())
		return
	}

	response := ProfessionalProjectToResponse(project)
	responses.Success(c, "Professional project updated successfully", response)
}

func (h *ProjectHandler) DeleteProfessionalProject(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		responses.BadRequest(c, "Invalid project ID")
		return
	}

	userID, exists := keycloakauth.GetUserID(c)
	if !exists {
		responses.Unauthorized(c, "User not authenticated")
		return
	}

	err = h.projectService.DeleteProfessionalProject(uint(id), userID)
	if err != nil {
		responses.InternalError(c, err.Error())
		return
	}

	responses.Success(c, "Professional project deleted successfully", nil)
}

func (h *ProjectHandler) GetUserProfessionalProjects(c *gin.Context) {
	userID, exists := keycloakauth.GetUserID(c)
	if !exists {
		responses.Unauthorized(c, "User not authenticated")
		return
	}

	projects, err := h.projectService.GetUserProfessionalProjects(userID)
	if err != nil {
		responses.InternalError(c, err.Error())
		return
	}

	projectResponses := ProfessionalProjectsToResponse(projects)
	responses.Success(c, "Professional projects retrieved successfully", gin.H{
		"projects": projectResponses,
		"total":    len(projectResponses),
	})
}

func (h *ProjectHandler) CreateFreelanceProject(c *gin.Context) {
	idParam := c.Param("id")
	parentProjectID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		responses.BadRequest(c, "Invalid parent project ID")
		return
	}

	var req CreateFreelanceProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.BadRequest(c, err.Error())
		return
	}

	userID, exists := keycloakauth.GetUserID(c)
	if !exists {
		responses.Unauthorized(c, "User not authenticated")
		return
	}

	freelanceProject := req.ToFreelanceProject()
	createdProject, err := h.projectService.CreateFreelanceProject(uint(parentProjectID), freelanceProject, userID)
	if err != nil {
		responses.InternalError(c, err.Error())
		return
	}

	response := FreelanceProjectToResponse(createdProject)
	responses.Created(c, "Freelance project created successfully", response)
}

func (h *ProjectHandler) GetFreelanceProject(c *gin.Context) {
	freelanceIDParam := c.Param("freelanceId")
	freelanceID, err := strconv.ParseUint(freelanceIDParam, 10, 32)
	if err != nil {
		responses.BadRequest(c, "Invalid freelance project ID")
		return
	}

	userID, exists := keycloakauth.GetUserID(c)
	if !exists {
		responses.Unauthorized(c, "User not authenticated")
		return
	}

	freelanceProject, err := h.projectService.GetFreelanceProject(uint(freelanceID), userID)
	if err != nil {
		if err.Error() == "access denied: freelance project is private to the worker" {
			responses.Forbidden(c, err.Error())
			return
		}
		responses.NotFound(c, err.Error())
		return
	}

	response := FreelanceProjectToResponse(freelanceProject)
	responses.Success(c, "Freelance project retrieved successfully", response)
}

func (h *ProjectHandler) UpdateFreelanceProject(c *gin.Context) {
	freelanceIDParam := c.Param("freelanceId")
	freelanceID, err := strconv.ParseUint(freelanceIDParam, 10, 32)
	if err != nil {
		responses.BadRequest(c, "Invalid freelance project ID")
		return
	}

	var req UpdateFreelanceProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.BadRequest(c, err.Error())
		return
	}

	userID, exists := keycloakauth.GetUserID(c)
	if !exists {
		responses.Unauthorized(c, "User not authenticated")
		return
	}

	updates := req.ToFreelanceProject()
	freelanceProject, err := h.projectService.UpdateFreelanceProject(uint(freelanceID), updates, userID)
	if err != nil {
		if err.Error() == "access denied: freelance project is private to the worker" {
			responses.Forbidden(c, err.Error())
			return
		}
		responses.InternalError(c, err.Error())
		return
	}

	response := FreelanceProjectToResponse(freelanceProject)
	responses.Success(c, "Freelance project updated successfully", response)
}

func (h *ProjectHandler) GetProjectCostReport(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		responses.BadRequest(c, "Invalid project ID")
		return
	}

	userID, exists := keycloakauth.GetUserID(c)
	if !exists {
		responses.Unauthorized(c, "User not authenticated")
		return
	}

	report, err := h.projectService.GetProjectCostReport(uint(id), userID)
	if err != nil {
		responses.InternalError(c, err.Error())
		return
	}

	responses.Success(c, "Project cost report generated successfully", report)
}
