package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestCloudServerBodyPerProvider(t *testing.T) {
	token, key := "tok1", "key1"
	location, serverType := "fsn1", "cx22"
	var image int64 = 114690389
	region, size, doImage := "fra1", "s-2vcpu-4gb", "ubuntu-24-04-x64"
	plan := "vc2-2c-4gb"
	var osID int64 = 2284

	hetzner, err := cloudServerBody("hetzner", CloudServerRequest{
		CloudProviderTokenUUID: &token, PrivateKeyUUID: &key,
		Location: &location, ServerType: &serverType, HetznerImage: &image,
	})
	if err != nil {
		t.Fatal(err)
	}
	if hetzner["image"] != image || hetzner["location"] != "fsn1" {
		t.Errorf("hetzner body = %v", hetzner)
	}

	do, err := cloudServerBody("digitalocean", CloudServerRequest{
		CloudProviderTokenUUID: &token, PrivateKeyUUID: &key,
		Region: &region, Size: &size, DigitalOceanImage: &doImage,
	})
	if err != nil {
		t.Fatal(err)
	}
	// DigitalOcean images are string slugs while Hetzner images are int ids.
	if do["image"] != "ubuntu-24-04-x64" || do["size"] != "s-2vcpu-4gb" {
		t.Errorf("digitalocean body = %v", do)
	}

	vultr, err := cloudServerBody("vultr", CloudServerRequest{
		CloudProviderTokenUUID: &token, PrivateKeyUUID: &key,
		Region: &region, Plan: &plan, OSID: &osID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if vultr["plan"] != "vc2-2c-4gb" || vultr["os_id"] != osID {
		t.Errorf("vultr body = %v", vultr)
	}

	if _, err := cloudServerBody("aws", CloudServerRequest{}); err == nil {
		t.Error("expected error for unsupported provider")
	}
}

func TestCreateCloudServerPostsAndReads(t *testing.T) {
	var posted map[string]any
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "POST /api/v1/servers/hetzner":
			_ = json.NewDecoder(r.Body).Decode(&posted)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"uuid":"srv9"}`))
		case "GET /api/v1/servers/srv9":
			_, _ = w.Write([]byte(`{"uuid":"srv9","name":"hetzner-1","ip":"1.2.3.4"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	token := "tok1"
	server, err := c.CreateCloudServer(context.Background(), "hetzner", CloudServerRequest{CloudProviderTokenUUID: &token})
	if err != nil {
		t.Fatalf("CreateCloudServer: %v", err)
	}
	if server.IP != "1.2.3.4" {
		t.Errorf("ip = %q", server.IP)
	}
	if posted["cloud_provider_token_uuid"] != "tok1" {
		t.Errorf("posted = %v", posted)
	}
}

func TestCloudCatalogPassesToken(t *testing.T) {
	var got string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.String()
		_, _ = w.Write([]byte(`[{"name":"fsn1","description":"Falkenstein"}]`))
	}))
	items, err := c.CloudCatalog(context.Background(), "hetzner", "locations", "tok1")
	if err != nil {
		t.Fatalf("CloudCatalog: %v", err)
	}
	if want := "/api/v1/hetzner/locations?cloud_provider_token_uuid=tok1"; got != want {
		t.Errorf("url = %q, want %q", got, want)
	}
	if len(items) != 1 || items[0]["name"] != "fsn1" {
		t.Errorf("items = %v", items)
	}
}

func TestCloudTokenLifecycle(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"uuid":"ct1"}`))
			return
		}
		_, _ = w.Write([]byte(`{"uuid":"ct1","name":"hetzner-prod","provider":"hetzner"}`))
	}))

	name, provider, token := "hetzner-prod", "hetzner", "secret"
	ct, err := c.CreateCloudToken(context.Background(), CloudTokenRequest{Name: &name, Provider: &provider, Token: &token})
	if err != nil {
		t.Fatalf("CreateCloudToken: %v", err)
	}
	if ct.Provider != "hetzner" {
		t.Errorf("provider = %q", ct.Provider)
	}
}
