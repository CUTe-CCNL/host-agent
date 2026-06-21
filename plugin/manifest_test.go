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
args: ["--addr", "127.0.0.1:19101"]
working_dir: "/opt/host-agent/plugins/firewall"
env:
  LOG_LEVEL: "info"
upstream_url: "http://127.0.0.1:19101"
health_path: "/health"
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
	if manifest.UpstreamURL != "http://127.0.0.1:19101" {
		t.Errorf("UpstreamURL = %s, want http://127.0.0.1:19101", manifest.UpstreamURL)
	}
	if len(manifest.Routes) != 1 {
		t.Fatalf("len(Routes) = %d, want 1", len(manifest.Routes))
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

func TestLoadManifestsRejectsNonLoopbackUpstream(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, "firewall.yaml", strings.Replace(validManifest("firewall"), "http://127.0.0.1:19101", "http://10.0.0.2:19101", 1))

	_, err := LoadManifests(dir)
	if err == nil {
		t.Fatal("LoadManifests() error = nil, want non-loopback upstream error")
	}
}

func TestLoadManifestsRejectsMissingRequiredFields(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, "firewall.yaml", `
id: "firewall"
name: "Firewall Manager"
version: "0.1.0"
enabled: true
upstream_url: "http://127.0.0.1:19101"
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
