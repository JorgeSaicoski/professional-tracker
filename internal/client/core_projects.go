package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

/* ---------- DTOs ---------- */

// we only care about the fields shown here; extend later if needed
type BaseProject struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	OwnerID   string  `json:"ownerId"`
	Status    string  `json:"status"`
	CompanyID *string `json:"companyId,omitempty"`
}

type BaseProjectCreateRequest struct {
	Title     string  `json:"title"`
	OwnerID   string  `json:"ownerId"`
	Status    string  `json:"status"`              // always "active"
	CompanyID *string `json:"companyId,omitempty"` // nil → empty in DB
}

/* ---------- Client interface & HTTP impl ---------- */

type CoreProjectClient interface {
	CreateBaseProject(ctx context.Context, req *BaseProjectCreateRequest) (*BaseProject, error)
}

type httpCoreProjectClient struct {
	baseURL string
	http    *http.Client
}

func NewCoreProjectHTTPClient(baseURL string) CoreProjectClient {
	return &httpCoreProjectClient{
		baseURL: baseURL,
		http:    &http.Client{},
	}
}

func (c *httpCoreProjectClient) CreateBaseProject(
	ctx context.Context,
	req *BaseProjectCreateRequest,
) (*BaseProject, error) {

	body, _ := json.Marshal(req)
	url := fmt.Sprintf("%s/projects", c.baseURL)

	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("core-project call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("core-project returned %s", resp.Status)
	}

	var bp BaseProject
	if err := json.NewDecoder(resp.Body).Decode(&bp); err != nil {
		return nil, fmt.Errorf("decode base project: %w", err)
	}
	return &bp, nil
}
