package projects

import (
	"strconv"

	keycloakauth "github.com/JorgeSaicoski/keycloak-auth"
	"github.com/JorgeSaicoski/microservice-commons/responses"
	"github.com/JorgeSaicoski/professional-tracker/internal/services/projects"
	"github.com/gin-gonic/gin"
)

/* ------------------------------------------------------------------ */
/*  Handler definition                                                */
/* ------------------------------------------------------------------ */

type ProjectHandler struct {
	projectService *projects.ProfessionalProjectService
}

func NewProjectHandler(svc *projects.ProfessionalProjectService) *ProjectHandler {
	return &ProjectHandler{projectService: svc}
}

/* ------------------------- Professional --------------------------- */

func (h *ProjectHandler) CreateProfessionalProject(c *gin.Context) {
	var req CreateProfessionalProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.BadRequest(c, err.Error())
		return
	}

	userID, ok := keycloakauth.GetUserID(c)
	if !ok {
		responses.Unauthorized(c, "User not authenticated")
		return
	}

	input := req.ToInput()
	created, err := h.projectService.CreateProfessionalProject(input, userID)
	if err != nil {
		responses.InternalError(c, err.Error())
		return
	}

	resp := ProfessionalProjectToResponse(created)
	responses.Created(c, "Professional project created successfully", resp)
}

/* ------------------------- R/W endpoints ------------------------- */

func (h *ProjectHandler) GetProfessionalProject(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		responses.BadRequest(c, "Invalid project ID")
		return
	}

	userID, ok := keycloakauth.GetUserID(c)
	if !ok {
		responses.Unauthorized(c, "User not authenticated")
		return
	}

	proj, err := h.projectService.GetProfessionalProject(uint(id), userID)
	if err != nil {
		responses.NotFound(c, err.Error())
		return
	}

	resp := ProfessionalProjectToResponse(proj)
	responses.Success(c, "Professional project retrieved successfully", resp)
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

	userID, ok := keycloakauth.GetUserID(c)
	if !ok {
		responses.Unauthorized(c, "User not authenticated")
		return
	}

	updates := req.ToProfessionalProject()
	proj, err := h.projectService.UpdateProfessionalProject(uint(id), updates, userID)
	if err != nil {
		responses.InternalError(c, err.Error())
		return
	}

	resp := ProfessionalProjectToResponse(proj)
	responses.Success(c, "Professional project updated successfully", resp)
}

func (h *ProjectHandler) DeleteProfessionalProject(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		responses.BadRequest(c, "Invalid project ID")
		return
	}

	userID, ok := keycloakauth.GetUserID(c)
	if !ok {
		responses.Unauthorized(c, "User not authenticated")
		return
	}

	if err := h.projectService.DeleteProfessionalProject(uint(id), userID); err != nil {
		responses.InternalError(c, err.Error())
		return
	}

	responses.Success(c, "Professional project deleted successfully", nil)
}

func (h *ProjectHandler) GetUserProfessionalProjects(c *gin.Context) {
	userID, ok := keycloakauth.GetUserID(c)
	if !ok {
		responses.Unauthorized(c, "User not authenticated")
		return
	}

	list, err := h.projectService.GetUserProfessionalProjects(userID)
	if err != nil {
		responses.InternalError(c, err.Error())
		return
	}

	resp := ProfessionalProjectsToResponse(list)
	responses.Success(c, "Professional projects retrieved successfully", gin.H{
		"projects": resp,
		"total":    len(resp),
	})
}

/* ------------------------- Freelance sub-projects ---------------- */

func (h *ProjectHandler) ProjectAssignment(c *gin.Context) {
	idParam := c.Param("id")
	parentID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		responses.BadRequest(c, "Invalid parent project ID")
		return
	}

	var req CreateProjectAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.BadRequest(c, err.Error())
		return
	}

	userID, ok := keycloakauth.GetUserID(c)
	if !ok {
		responses.Unauthorized(c, "User not authenticated")
		return
	}

	fp := req.ToProjectAssignment()
	created, err := h.projectService.ProjectAssignment(uint(parentID), fp, userID)
	if err != nil {
		responses.InternalError(c, err.Error())
		return
	}

	resp := ProjectAssignmentToResponse(created)
	responses.Created(c, "Freelance project created successfully", resp)
}

func (h *ProjectHandler) GetProjectAssignment(c *gin.Context) {
	fidParam := c.Param("freelanceId")
	fid, err := strconv.ParseUint(fidParam, 10, 32)
	if err != nil {
		responses.BadRequest(c, "Invalid freelance project ID")
		return
	}

	userID, ok := keycloakauth.GetUserID(c)
	if !ok {
		responses.Unauthorized(c, "User not authenticated")
		return
	}

	fp, err := h.projectService.GetProjectAssignment(uint(fid), userID)
	if err != nil {
		if err.Error() == "access denied: freelance project is private to the worker" {
			responses.Forbidden(c, err.Error())
			return
		}
		responses.NotFound(c, err.Error())
		return
	}

	resp := ProjectAssignmentToResponse(fp)
	responses.Success(c, "Freelance project retrieved successfully", resp)
}

func (h *ProjectHandler) UpdateProjectAssignment(c *gin.Context) {
	fidParam := c.Param("freelanceId")
	fid, err := strconv.ParseUint(fidParam, 10, 32)
	if err != nil {
		responses.BadRequest(c, "Invalid freelance project ID")
		return
	}

	var req UpdateProjectAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.BadRequest(c, err.Error())
		return
	}

	userID, ok := keycloakauth.GetUserID(c)
	if !ok {
		responses.Unauthorized(c, "User not authenticated")
		return
	}

	updates := req.ToProjectAssignment()
	fp, err := h.projectService.UpdateProjectAssignment(uint(fid), updates, userID)
	if err != nil {
		if err.Error() == "access denied: freelance project is private to the worker" {
			responses.Forbidden(c, err.Error())
			return
		}
		responses.InternalError(c, err.Error())
		return
	}

	resp := ProjectAssignmentToResponse(fp)
	responses.Success(c, "Freelance project updated successfully", resp)
}

/* ------------------------- Reports ------------------------------- */

func (h *ProjectHandler) GetProjectCostReport(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		responses.BadRequest(c, "Invalid project ID")
		return
	}

	userID, ok := keycloakauth.GetUserID(c)
	if !ok {
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
