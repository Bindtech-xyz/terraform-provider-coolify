package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
)

// DefaultServiceTemplatesURL is where Coolify itself pulls its one-click
// service catalog from. Fetching it at plan time keeps the provider in sync
// with new services without a provider release.
const DefaultServiceTemplatesURL = "https://cdn.coollabs.io/coolify/service-templates-latest.json"

// ServiceTemplate describes one entry of the one-click service catalog.
type ServiceTemplate struct {
	Type          string `json:"-"` // map key, e.g. "plausible"
	Slogan        string `json:"slogan"`
	Category      string `json:"category"`
	Documentation string `json:"documentation"`
	Port          string `json:"port"`
	Logo          string `json:"logo"`
	MinVersion    string `json:"minversion"`
}

// FetchServiceTemplates downloads the service catalog. The URL is
// unauthenticated and independent of the configured Coolify instance, so this
// uses the underlying HTTP client without the bearer token. Templates are
// returned sorted by type.
func (c *Client) FetchServiceTemplates(ctx context.Context, templatesURL string) ([]ServiceTemplate, error) {
	if templatesURL == "" {
		templatesURL = DefaultServiceTemplatesURL
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, templatesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", templatesURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: HTTP %d", templatesURL, resp.StatusCode)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("GET %s: reading response: %w", templatesURL, err)
	}

	var decoded map[string]ServiceTemplate
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("GET %s: decoding catalog: %w", templatesURL, err)
	}

	out := make([]ServiceTemplate, 0, len(decoded))
	for name, tpl := range decoded {
		tpl.Type = name
		out = append(out, tpl)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Type < out[j].Type })
	return out, nil
}
