package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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

/* ---------------------------------------------------------------------
   Client interface & HTTP implementation
   ------------------------------------------------------------------ */

type CoreProjectClient interface {
	CreateBaseProject(ctx context.Context, req *BaseProjectCreateRequest) (*BaseProject, error)
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
