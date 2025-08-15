package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

/* ---------------------------------------------------------------------
   Data‑transfer objects (DTOs)
   ---------------------------------------------------------------------
   These structs describe the JSON contract that we exchange with the
   Core‑Projects service.  We purposely expose **only** the fields we use
   today; adding a new JSON tag later is backward‑compatible.
   ------------------------------------------------------------------ */

type BaseProject struct {
	ID        string  `json:"id"` // ← primary‑key returned by Core; string for easy cross‑service use
	Title     string  `json:"title"`
	OwnerID   string  `json:"ownerId"`
	Status    string  `json:"status"`
	CompanyID *string `json:"companyId,omitempty"` // pointer → field omitted when nil
}

type BaseProjectCreateRequest struct {
	Title     string  `json:"title"`
	OwnerID   string  `json:"ownerId"`
	Status    string  `json:"status"`              // always "active" in current workflow
	CompanyID *string `json:"companyId,omitempty"` // optional FK
}

// Members returned by Core.
type ProjectMember struct {
	ProjectID   string    `json:"projectId"`
	ProjectType string    `json:"projectType,omitempty"`
	UserID      string    `json:"userId"`
	Role        string    `json:"role"`
	Permissions []string  `json:"permissions,omitempty"`
	JoinedAt    time.Time `json:"joinedAt"`
}

// Update payload for Core.
type UpdateProjectRequest struct {
	Title       string     `json:"title,omitempty"`
	Description *string    `json:"description,omitempty"`
	Status      string     `json:"status,omitempty"`
	StartDate   *time.Time `json:"startDate,omitempty"`
	EndDate     *time.Time `json:"endDate,omitempty"`
}

// Add-member payload for Core.
type AddMemberRequest struct {
	UserID           string   `json:"userId"`
	Role             string   `json:"role"`
	Permissions      []string `json:"permissions,omitempty"`
	RequestingUserID string   `json:"requestingUserId"`
}

/* ---------------------------------------------------------------------
   Client interface & HTTP implementation
   ------------------------------------------------------------------ */

type CoreProjectClient interface {
	CreateBaseProject(ctx context.Context, req *BaseProjectCreateRequest) (*BaseProject, error)
	GetProject(ctx context.Context, id string, userID string) (*BaseProject, error)
	UpdateProject(ctx context.Context, id string, userID string, updates *UpdateProjectRequest) (*BaseProject, error)
	DeleteProject(ctx context.Context, id string, userID string) error
	GetUserProjects(ctx context.Context, userID string) ([]BaseProject, error)
	GetProjectMembers(ctx context.Context, id string, userID string) ([]ProjectMember, error)
	AddProjectMember(ctx context.Context, id string, req *AddMemberRequest) (*ProjectMember, error)
}

// httpCoreProjectClient is a thin wrapper over net/http that knows how to
// talk to the Core‑Projects service.  We keep it dumb on purpose: it builds
// the request, unmarshals the response, nothing more.

type httpCoreProjectClient struct {
	baseURL string       // e.g. "http://project-core:8080/api/internal"
	http    *http.Client // injected so we can swap in mocks/timeouts later
}

// NewCoreProjectHTTPClient is the public constructor used by the
// Professional‑Tracker service at boot time.
func NewCoreProjectHTTPClient(baseURL string) CoreProjectClient {
	return &httpCoreProjectClient{
		baseURL: baseURL,
		http:    &http.Client{},
	}
}

/* ---------------------------------------------------------------------
   CreateBaseProject – POST /projects
   ------------------------------------------------------------------ */

func (c *httpCoreProjectClient) CreateBaseProject(
	ctx context.Context,
	req *BaseProjectCreateRequest,
) (*BaseProject, error) {

	// 1)  Encode request object → JSON body
	body, _ := json.Marshal(req)
	url := fmt.Sprintf("%s/projects", c.baseURL)

	// 2)  Build the HTTP request with the caller‑supplied context so the
	//     caller can cancel / set timeouts.
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	// 3)  Execute the request.
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("core-project call failed: %w", err)
	}
	defer resp.Body.Close()

	// 4)  Non‑2xx → bubble up the plain body for easier troubleshooting.
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("core-project returned %s – body: %s", resp.Status, raw)
	}

	// 5)  Read whole body so we can both debug‑print and unmarshal.
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	log.Printf("[DEBUG] Core response: %s", raw)

	/*
	   Core responds with an envelope like:
	   {
	     "message": "Project created successfully",
	     "data": {
	         "id": 6,
	         "title": "admin",
	         ...
	     },
	     "timestamp": "2025-07-30T18:06:19Z"
	   }

	   We care only about the object inside "data", and we must tolerate
	   `id` arriving as either a JSON number (6) or string ("6").
	*/

	type envelope struct {
		Data struct {
			ID      json.Number `json:"id"` // json.Number → handles both number and string
			Title   string      `json:"title"`
			OwnerID string      `json:"ownerId"`
		} `json:"data"`
	}

	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("decode base project: %w", err)
	}

	// 6)  Promote the inner struct to our public DTO, string‑ifying the ID.
	bp := &BaseProject{
		ID:      env.Data.ID.String(), // json.Number.String() gives the textual form (e.g. "6")
		Title:   env.Data.Title,
		OwnerID: env.Data.OwnerID,
	}
	return bp, nil
}

/*
---------------------------------------------------------------------

	GetProject – GET {baseURL}/projects/:id
	Header: X-User-ID
	Envelope: { "data": { "id": number|string, ... } }
	------------------------------------------------------------------
*/
func (c *httpCoreProjectClient) GetProject(ctx context.Context, id string, userID string) (*BaseProject, error) {
	u := fmt.Sprintf("%s/projects/%s", c.baseURL, id)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	req.Header.Set("X-User-ID", userID)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("core-project get: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("core-project get %s: %s", resp.Status, body)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	var env struct {
		Data struct {
			ID        json.Number `json:"id"`
			Title     string      `json:"title"`
			OwnerID   string      `json:"ownerId"`
			Status    string      `json:"status"`
			CompanyID *string     `json:"companyId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("decode get project: %w", err)
	}
	return &BaseProject{
		ID:        env.Data.ID.String(),
		Title:     env.Data.Title,
		OwnerID:   env.Data.OwnerID,
		Status:    env.Data.Status,
		CompanyID: env.Data.CompanyID,
	}, nil
}

/*
---------------------------------------------------------------------

	UpdateProject – PUT {baseURL}/projects/:id
	Body: { ...updates, userId }
	Envelope: { "data": { ... } }
	------------------------------------------------------------------
*/
func (c *httpCoreProjectClient) UpdateProject(ctx context.Context, id string, userID string, updates *UpdateProjectRequest) (*BaseProject, error) {
	u := fmt.Sprintf("%s/projects/%s", c.baseURL, id)
	payload := struct {
		*UpdateProjectRequest
		UserID string `json:"userId"`
	}{updates, userID}
	raw, _ := json.Marshal(payload)

	req, _ := http.NewRequestWithContext(ctx, http.MethodPut, u, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("core-project update: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("core-project update %s: %s", resp.Status, body)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	var env struct {
		Data struct {
			ID        json.Number `json:"id"`
			Title     string      `json:"title"`
			OwnerID   string      `json:"ownerId"`
			Status    string      `json:"status"`
			CompanyID *string     `json:"companyId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("decode update project: %w", err)
	}
	return &BaseProject{
		ID:        env.Data.ID.String(),
		Title:     env.Data.Title,
		OwnerID:   env.Data.OwnerID,
		Status:    env.Data.Status,
		CompanyID: env.Data.CompanyID,
	}, nil
}

/*
---------------------------------------------------------------------

	DeleteProject – DELETE {baseURL}/projects/:id
	Header: X-User-ID
	------------------------------------------------------------------
*/
func (c *httpCoreProjectClient) DeleteProject(ctx context.Context, id string, userID string) error {
	u := fmt.Sprintf("%s/projects/%s", c.baseURL, id)
	req, _ := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	req.Header.Set("X-User-ID", userID)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("core-project delete: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("core-project delete %s: %s", resp.Status, body)
	}
	return nil
}

/*
---------------------------------------------------------------------

	GetUserProjects – GET {baseURL}/projects?userId=...
	Envelope: { "data": { "data": [ ... ] } }
	------------------------------------------------------------------
*/
func (c *httpCoreProjectClient) GetUserProjects(ctx context.Context, userID string) ([]BaseProject, error) {
	u := fmt.Sprintf("%s/projects?userId=%s", c.baseURL, url.QueryEscape(userID))
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("core-project list: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("core-project list %s: %s", resp.Status, body)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	var env struct {
		Data struct {
			Data []struct {
				ID        json.Number `json:"id"`
				Title     string      `json:"title"`
				OwnerID   string      `json:"ownerId"`
				Status    string      `json:"status"`
				CompanyID *string     `json:"companyId"`
			} `json:"data"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("decode list projects: %w", err)
	}
	out := make([]BaseProject, 0, len(env.Data.Data))
	for _, p := range env.Data.Data {
		out = append(out, BaseProject{
			ID:        p.ID.String(),
			Title:     p.Title,
			OwnerID:   p.OwnerID,
			Status:    p.Status,
			CompanyID: p.CompanyID,
		})
	}
	return out, nil
}

/*
---------------------------------------------------------------------

	GetProjectMembers – GET {baseURL}/projects/:id/members?userId=...
	Envelope: { "data": { "members": [ ... ], "total": n } }
	------------------------------------------------------------------
*/
func (c *httpCoreProjectClient) GetProjectMembers(ctx context.Context, id string, userID string) ([]ProjectMember, error) {
	u := fmt.Sprintf("%s/projects/%s/members?userId=%s", c.baseURL, id, url.QueryEscape(userID))
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("core-project members: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("core-project members %s: %s", resp.Status, body)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	var env struct {
		Data struct {
			Members []ProjectMember `json:"members"`
			Total   int             `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("decode members: %w", err)
	}
	return env.Data.Members, nil
}

/*
---------------------------------------------------------------------

	AddProjectMember – POST {baseURL}/projects/:id/members
	Body: { userId, role, permissions, requestingUserId }
	Envelope: { "data": { ...member } }
	------------------------------------------------------------------
*/
func (c *httpCoreProjectClient) AddProjectMember(ctx context.Context, id string, reqBody *AddMemberRequest) (*ProjectMember, error) {
	u := fmt.Sprintf("%s/projects/%s/members", c.baseURL, id)
	raw, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("core-project add member: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("core-project add member %s: %s", resp.Status, body)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	var env struct {
		Data ProjectMember `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("decode add member: %w", err)
	}
	return &env.Data, nil
}
