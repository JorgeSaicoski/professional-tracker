package projects

import (
	"bytes"
	"io"
	"log"
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

func (h *ProjectHandler) CreateProjectAssignment(c *gin.Context) {
	// 1. Log entry point.
	log.Println("DEBUG: Entering CreateProjectAssignment handler")

	// 2. Read and log the entire raw request body.
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("ERROR: Failed to read request body: %v", err)
		responses.InternalError(c, "Failed to read request body")
		return
	}
	log.Printf("DEBUG: Raw request body: %s", string(bodyBytes))

	// 3. Restore the request body for subsequent Gin operations.
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// The rest of your existing code follows...

	idParam := c.Param("id")
	log.Printf("DEBUG: idParam from URL: %s", idParam)
	parentID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		log.Printf("ERROR: Failed to parse parentID from URL param '%s': %v", idParam, err)
		responses.BadRequest(c, "Invalid parent project ID")
		return
	}
	log.Printf("DEBUG: Successfully parsed parentID: %d", parentID)

	var req CreateProjectAssignmentRequest
	// This will now successfully bind from the restored body.
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("ERROR: Failed to bind JSON request body: %v", err)
		responses.BadRequest(c, err.Error())
		return
	}
	log.Printf("DEBUG: Successfully bound JSON request: %+v", req)

	userID, ok := keycloakauth.GetUserID(c)
	if !ok {
		log.Println("ERROR: User not authenticated from Keycloak")
		responses.Unauthorized(c, "User not authenticated")
		return
	}
	log.Printf("DEBUG: Authenticated userID: %s", userID)

	fp := req.ToProjectAssignment()
	created, err := h.projectService.CreateProjectAssignment(uint(parentID), fp, userID)
	if err != nil {
		log.Printf("ERROR: Service failed to create project assignment: %v", err)
		responses.InternalError(c, err.Error())
		return
	}

	log.Printf("DEBUG: Successfully created project assignment with ID: %d", created.ID)
	resp := ProjectAssignmentToResponse(created)
	log.Printf("DEBUG: Final response payload: %+v", resp)
	responses.Created(c, "Freelance project created successfully", resp)

	log.Println("DEBUG: Exiting CreateProjectAssignment handler")
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

// GetMyAssignments returns all ProjectAssignments where the caller is the worker.
// Auth is enforced by middleware; this reads user ID from header set by your gateway/middleware.
func (h *ProjectHandler) GetMyAssignments(c *gin.Context) {
	userID, ok := keycloakauth.GetUserID(c)
	if !ok {
		responses.Unauthorized(c, "User not authenticated")
		return
	}

	assignments, err := h.projectService.GetUserProjectAssignmentsCtx(c.Request.Context(), userID)
	if err != nil {
		responses.InternalError(c, err.Error())
		return
	}

	out := make([]*ProfessionalAssignmentDTO, 0, len(assignments))
	for _, a := range assignments {
		out = append(out, NewProfessionalAssignmentDTO(a))
	}
	responses.Success(c, "Assignments retrieved successfully", out)
}
