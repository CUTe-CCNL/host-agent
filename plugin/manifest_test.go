package plugin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeManifest(t *testing.T, dir, name, content string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write manifest %s: %v", name, err)
	}
}

func validManifest(id string) string {
	return `
id: "` + id + `"
name: "Firewall Manager"
version: "0.1.0"
enabled: true
description: "Manage local firewall rules"
command: "/bin/host-agent-firewall"
args: ["--stdio"]
working_dir: "/opt/host-agent/plugins/firewall"
env:
  LOG_LEVEL: "info"
routes:
  - path_prefix: "/"
    methods: ["GET", "POST", "PUT", "PATCH", "DELETE"]
`
}

func TestLoadManifestsLoadsValidManifest(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, "firewall.yaml", validManifest("firewall"))

	manifests, err := LoadManifests(dir)
	if err != nil {
		t.Fatalf("LoadManifests() error = %v", err)
	}

	if len(manifests) != 1 {
		t.Fatalf("len(manifests) = %d, want 1", len(manifests))
	}

	manifest := manifests[0]
	if manifest.ID != "firewall" {
		t.Errorf("ID = %s, want firewall", manifest.ID)
	}
	if !manifest.Enabled {
		t.Error("Enabled should be true")
	}
	if len(manifest.Routes) != 1 {
		t.Fatalf("len(Routes) = %d, want 1", len(manifest.Routes))
	}
}

func TestLoadManifestsLoadsDocsExampleManifest(t *testing.T) {
	manifests, err := LoadManifests(filepath.Join("..", "docs", "plugins", "example-go"))
	if err != nil {
		t.Fatalf("LoadManifests() error = %v", err)
	}

	if len(manifests) != 1 {
		t.Fatalf("len(manifests) = %d, want 1", len(manifests))
	}

	manifest := manifests[0]
	if manifest.ID != "example-go" {
		t.Errorf("ID = %s, want example-go", manifest.ID)
	}
	if manifest.Command != "/opt/host-agent/plugins/example-go/example-go-plugin" {
		t.Errorf("Command = %s, want /opt/host-agent/plugins/example-go/example-go-plugin", manifest.Command)
	}
	wantRoutes := map[string][]string{
		"/status": {"GET"},
		"/echo":   {"GET", "POST"},
		"/items":  {"GET", "POST", "DELETE"},
	}
	if len(manifest.Routes) != len(wantRoutes) {
		t.Fatalf("len(Routes) = %d, want %d", len(manifest.Routes), len(wantRoutes))
	}
	for _, route := range manifest.Routes {
		wantMethods, ok := wantRoutes[route.PathPrefix]
		if !ok {
			t.Fatalf("unexpected route path_prefix %q", route.PathPrefix)
		}
		if strings.Join(route.Methods, ",") != strings.Join(wantMethods, ",") {
			t.Fatalf("route %s methods = %v, want %v", route.PathPrefix, route.Methods, wantMethods)
		}
	}
}

func TestLoadManifestsRejectsDuplicateID(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, "one.yaml", validManifest("firewall"))
	writeManifest(t, dir, "two.yaml", validManifest("firewall"))

	_, err := LoadManifests(dir)
	if err == nil {
		t.Fatal("LoadManifests() error = nil, want duplicate id error")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("error = %q, want duplicate id", err.Error())
	}
}

func TestLoadManifestsRejectsInvalidMethod(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, "firewall.yaml", strings.Replace(validManifest("firewall"), `"GET", "POST", "PUT", "PATCH", "DELETE"`, `"GET", "TRACE"`, 1))

	_, err := LoadManifests(dir)
	if err == nil {
		t.Fatal("LoadManifests() error = nil, want invalid method error")
	}
}

func TestLoadManifestsRejectsMissingRequiredFields(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, "firewall.yaml", `
id: "firewall"
name: "Firewall Manager"
version: "0.1.0"
enabled: true
routes:
  - path_prefix: "/"
    methods: ["GET"]
`)

	_, err := LoadManifests(dir)
	if err == nil {
		t.Fatal("LoadManifests() error = nil, want missing command error")
	}
}

func TestLoadManifestsRejectsInvalidSlug(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, "firewall.yaml", validManifest("bad_id"))

	_, err := LoadManifests(dir)
	if err == nil {
		t.Fatal("LoadManifests() error = nil, want invalid slug error")
	}
}
