package schemaregistry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetSchemaURIUsesSchemaDirectory(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	schemaDir := filepath.Join(root, "schemas")
	cacheDir := filepath.Join(root, "generated")
	registry, err := NewRegistry(schemaDir, cacheDir)
	if err != nil {
		t.Fatal(err)
	}

	schemaPath := filepath.Join("kubernetes-builtin", "v1.36.0-standalone-strict", "service-v1.json")
	fullPath := filepath.Join(schemaDir, schemaPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte(`{"type":"object"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	uri, err := registry.GetSchemaURI("http://127.0.0.1:1/should-not-be-requested", schemaPath)
	if err != nil {
		t.Fatal(err)
	}
	if want := "file://" + fullPath; uri != want {
		t.Fatalf("GetSchemaURI() = %q, want %q", uri, want)
	}
}

func TestGenerateCompositeSchemaUsesSeparateCacheDirectory(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	schemaDir := filepath.Join(root, "schemas")
	cacheDir := filepath.Join(root, "generated")
	registry, err := NewRegistry(schemaDir, cacheDir)
	if err != nil {
		t.Fatal(err)
	}

	uri, err := registry.GenerateCompositeSchema([]string{
		"file:///schemas/deployment.json",
		"file:///schemas/service.json",
	})
	if err != nil {
		t.Fatal(err)
	}

	if wantPrefix := "file://" + filepath.Join(cacheDir, "composite") + string(filepath.Separator); !strings.HasPrefix(uri, wantPrefix) {
		t.Fatalf("GenerateCompositeSchema() = %q, want prefix %q", uri, wantPrefix)
	}
	if _, err := os.Stat(strings.TrimPrefix(uri, "file://")); err != nil {
		t.Fatalf("generated composite does not exist: %v", err)
	}
	entries, err := os.ReadDir(schemaDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("schema directory contains generated entries: %v", entries)
	}
}
