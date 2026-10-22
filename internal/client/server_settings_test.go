package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"
)

// TestSaveServerProxyConfigurationBase64EncodesBody locks in the API's
// documented convention (confirmed against ServerProxyController::
// saveConfiguration — isBase64Encoded()/base64_decode()): multi-line
// configuration must be sent base64-encoded, not as raw YAML, or the server
// would have to guess which form it received.
func TestSaveServerProxyConfigurationBase64EncodesBody(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]any
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"message":"Proxy configuration saved."}`))
	}))

	yaml := "services:\n  traefik:\n    container_name: coolify-proxy\n"
	if err := c.SaveServerProxyConfiguration(context.Background(), "srv1", yaml); err != nil {
		t.Fatalf("SaveServerProxyConfiguration: %v", err)
	}

	if want := "/api/v1/servers/srv1/proxy/configuration"; gotPath != want {
		t.Errorf("path = %q, want %q", gotPath, want)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	encoded, ok := gotBody["configuration"].(string)
	if !ok {
		t.Fatalf("body = %+v, missing configuration", gotBody)
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || string(decoded) != yaml {
		t.Errorf("configuration decoded = %q (err %v), want %q", decoded, err, yaml)
	}
}

// TestRestartServerProxyPath locks in the restart endpoint's path/method.
func TestRestartServerProxyPath(t *testing.T) {
	var gotPath, gotMethod string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		_, _ = w.Write([]byte(`{"message":"Proxy restart queued."}`))
	}))

	if err := c.RestartServerProxy(context.Background(), "srv1"); err != nil {
		t.Fatalf("RestartServerProxy: %v", err)
	}
	if want := "/api/v1/servers/srv1/proxy/restart"; gotPath != want {
		t.Errorf("path = %q, want %q", gotPath, want)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
}
