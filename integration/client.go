// Package integration is the supported Go client for independently deployed
// ApexVoid external applications. It intentionally exposes only the versioned
// service-to-platform protocol, never database or platform implementation APIs.
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	ContractVersion         = "v1"
	IdentityAssertionHeader = "X-ApexVoid-Identity-Assertion"
	ApplicationIDHeader     = "X-ApexVoid-Application-ID"
	ServiceCredentialHeader = "X-ApexVoid-Service-Credential"
)

type Config struct {
	PlatformURL       string
	ApplicationID     string
	ServiceCredential string
	HTTPClient        *http.Client
	Timeout           time.Duration
	RetryAttempts     int
}

type Client struct {
	platformURL   *url.URL
	applicationID string
	credential    string
	httpClient    *http.Client
	retries       int
}

type Decision struct {
	UserID      string `json:"user_id"`
	WorkspaceID string `json:"workspace_id"`
	Permission  string `json:"permission"`
	Allowed     bool   `json:"allowed"`
}

type Availability struct {
	ApplicationID string `json:"application_id"`
	WorkspaceID   string `json:"workspace_id"`
	Enabled       bool   `json:"enabled"`
}

// Error is safe to return or log. It deliberately never stores request
// headers, credentials, tokens, or raw assertions.
type Error struct {
	StatusCode int
	Code       string
	Message    string
	RequestID  string
}

func (e *Error) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("ApexVoid integration request failed: %s", e.Code)
	}
	return fmt.Sprintf("ApexVoid integration request failed with status %d", e.StatusCode)
}

func NewClient(config Config) (*Client, error) {
	endpoint, err := url.Parse(strings.TrimRight(strings.TrimSpace(config.PlatformURL), "/"))
	if err != nil || endpoint.Scheme == "" || endpoint.Host == "" {
		return nil, errors.New("integration platform URL must be an absolute HTTP URL")
	}
	if endpoint.Scheme != "http" && endpoint.Scheme != "https" {
		return nil, errors.New("integration platform URL must use HTTP or HTTPS")
	}
	if strings.TrimSpace(config.ApplicationID) == "" || strings.TrimSpace(config.ServiceCredential) == "" {
		return nil, errors.New("integration application ID and service credential are required")
	}
	client := config.HTTPClient
	if client == nil {
		timeout := config.Timeout
		if timeout <= 0 {
			timeout = 5 * time.Second
		}
		client = &http.Client{Timeout: timeout}
	}
	retries := config.RetryAttempts
	if retries < 0 {
		retries = 0
	}
	if retries > 2 {
		retries = 2
	}
	return &Client{platformURL: endpoint, applicationID: config.ApplicationID, credential: config.ServiceCredential, httpClient: client, retries: retries}, nil
}

func IdentityAssertionFromRequest(request *http.Request) (string, error) {
	if request == nil {
		return "", errors.New("integration request is required")
	}
	assertion := strings.TrimSpace(request.Header.Get(IdentityAssertionHeader))
	if assertion == "" {
		return "", errors.New("gateway identity assertion is required")
	}
	return assertion, nil
}

func (c *Client) Introspect(ctx context.Context, identityAssertion, permission string) (Decision, error) {
	if strings.TrimSpace(identityAssertion) == "" || strings.TrimSpace(permission) == "" {
		return Decision{}, errors.New("identity assertion and permission are required")
	}
	var decision Decision
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/integrations/"+ContractVersion+"/session/introspect", map[string]string{"identity_assertion": identityAssertion, "permission": permission}, &decision)
	return decision, err
}

func (c *Client) Availability(ctx context.Context, workspaceID string) (Availability, error) {
	if strings.TrimSpace(workspaceID) == "" {
		return Availability{}, errors.New("workspace ID is required")
	}
	var availability Availability
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/integrations/"+ContractVersion+"/applications/"+url.PathEscape(c.applicationID)+"/availability?workspace_id="+url.QueryEscape(workspaceID), nil, &availability)
	return availability, err
}

func (c *Client) doJSON(ctx context.Context, method, path string, body any, output any) error {
	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	for attempt := 0; ; attempt++ {
		request, err := http.NewRequestWithContext(ctx, method, c.platformURL.String()+path, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		request.Header.Set("Accept", "application/json")
		request.Header.Set(ApplicationIDHeader, c.applicationID)
		request.Header.Set(ServiceCredentialHeader, c.credential)
		if body != nil {
			request.Header.Set("Content-Type", "application/json")
		}
		response, requestErr := c.httpClient.Do(request)
		if requestErr == nil {
			if response.StatusCode >= 200 && response.StatusCode < 300 {
				defer response.Body.Close()
				if output == nil || response.StatusCode == http.StatusNoContent {
					return nil
				}
				return json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(output)
			}
			apiErr := decodeError(response)
			_ = response.Body.Close()
			if !retryableStatus(apiErr.StatusCode) || attempt == c.retries {
				return apiErr
			}
		} else if attempt == c.retries {
			return requestErr
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(attempt+1) * 100 * time.Millisecond):
		}
	}
}

func decodeError(response *http.Response) *Error {
	result := &Error{StatusCode: response.StatusCode}
	var body struct {
		Error struct {
			Code      string `json:"code"`
			Message   string `json:"message"`
			RequestID string `json:"request_id"`
		} `json:"error"`
	}
	if json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&body) == nil {
		result.Code, result.Message, result.RequestID = body.Error.Code, body.Error.Message, body.Error.RequestID
	}
	return result
}

func retryableStatus(status int) bool {
	return status == http.StatusBadGateway || status == http.StatusServiceUnavailable || status == http.StatusGatewayTimeout
}
