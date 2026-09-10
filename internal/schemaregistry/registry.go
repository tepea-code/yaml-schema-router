package schemaregistry

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"go.trai.ch/yaml-schema-router/internal/config"
)

const componentName = "Registry"

// ErrSchemaUnavailableOffline indicates that offline mode prevented a missing schema download.
var ErrSchemaUnavailableOffline = errors.New("schema is unavailable in the local schema directory")

// Registry manages local schemas and a separate cache for generated schemas.
type Registry struct {
	schemaDir string
	cacheDir  string
	offline   bool
}

type compositeSchema struct {
	AnyOf []map[string]string `json:"anyOf"`
}

// NewRegistry initializes the local schema and generated-schema directories.
func NewRegistry(schemaDir, cacheDir string, offline bool) (*Registry, error) {
	if err := os.MkdirAll(schemaDir, config.DefaultDirPerm); err != nil {
		return nil, fmt.Errorf("could not create schema dir: %w", err)
	}
	if err := os.MkdirAll(cacheDir, config.DefaultDirPerm); err != nil {
		return nil, fmt.Errorf("could not create generated schema cache dir: %w", err)
	}

	return &Registry{
		schemaDir: schemaDir,
		cacheDir:  cacheDir,
		offline:   offline,
	}, nil
}

// GetSchemaURI checks if the schema exists on disk. If not, it attempts to
// download it. Returns a file:// URI on success, or an error if it fails.
func (r *Registry) GetSchemaURI(remoteURL, cachePath string) (string, error) {
	fullPath := filepath.Join(r.schemaDir, cachePath)

	if _, err := os.Stat(fullPath); err == nil {
		log.Printf("[%s] Cache hit: %s", componentName, cachePath)
		return fmt.Sprintf("file://%s", fullPath), nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("failed to inspect local schema %s: %w", fullPath, err)
	}

	if r.offline {
		return "", fmt.Errorf("%w: %s", ErrSchemaUnavailableOffline, fullPath)
	}

	log.Printf("[%s] Cache miss: %s. Downloading from %s ...", componentName, cachePath, remoteURL)

	// Cache miss: download the schema
	data, err := download(remoteURL)
	if err != nil {
		// Return the error instead of falling back blindly
		return "", fmt.Errorf("failed to download %s: %w", remoteURL, err)
	}

	log.Printf("[%s] Download successful. Saving to %s", componentName, fullPath)

	// Save to cache
	if err := r.saveSchema(cachePath, data); err != nil {
		return "", fmt.Errorf("failed to save %s: %w", fullPath, err)
	}

	return fmt.Sprintf("file://%s", fullPath), nil
}

// saveSchema writes a downloaded schema to the local schema directory.
func (r *Registry) saveSchema(schemaPath string, data []byte) error {
	fullPath := filepath.Join(r.schemaDir, schemaPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), config.DefaultDirPerm); err != nil {
		return err
	}

	return os.WriteFile(fullPath, data, config.DefaultFilePerm)
}

// GetGeneratedPath returns the absolute path for a generated schema cache path.
func (r *Registry) GetGeneratedPath(cachePath string) string {
	return filepath.Join(r.cacheDir, cachePath)
}

// SaveGeneratedSchema writes a generated schema to the generated-schema cache.
func (r *Registry) SaveGeneratedSchema(cachePath string, data []byte) error {
	fullPath := r.GetGeneratedPath(cachePath)
	if err := os.MkdirAll(filepath.Dir(fullPath), config.DefaultDirPerm); err != nil {
		return err
	}

	return os.WriteFile(fullPath, data, config.DefaultFilePerm)
}

// GetGeneratedFileURI returns the file URI for a generated schema cache path.
func (r *Registry) GetGeneratedFileURI(cachePath string) string {
	return fmt.Sprintf("file://%s", r.GetGeneratedPath(cachePath))
}

// GenerateCompositeSchema creates a single schema using 'anyOf' to aggregate multiple schemas.
func (r *Registry) GenerateCompositeSchema(schemaURIs []string) (string, error) {
	if len(schemaURIs) == 0 {
		return "", nil
	}
	// Fast path: if there is only one manifest, just return its schema directly.
	if len(schemaURIs) == 1 {
		return schemaURIs[0], nil
	}

	// Deduplicate and sort URIs to ensure a deterministic hash for identical documents
	uriMap := make(map[string]bool)
	var uniqueURIs []string
	for _, uri := range schemaURIs {
		if !uriMap[uri] {
			uriMap[uri] = true
			uniqueURIs = append(uniqueURIs, uri)
		}
	}
	sort.Strings(uniqueURIs)

	// Hash the sorted URIs
	hash := sha256.New()
	for _, uri := range uniqueURIs {
		hash.Write([]byte(uri))
	}
	hashStr := hex.EncodeToString(hash.Sum(nil))[:16]

	cachePath := filepath.Join("composite", fmt.Sprintf("composite_%s.json", hashStr))

	// Fast path: check if this exact composite combination already exists
	if _, err := os.Stat(r.GetGeneratedPath(cachePath)); err == nil {
		return r.GetGeneratedFileURI(cachePath), nil
	}

	// Build the wrapper
	anyOf := make([]map[string]string, 0, len(uniqueURIs))
	for _, uri := range uniqueURIs {
		anyOf = append(anyOf, map[string]string{"$ref": uri})
	}

	composite := compositeSchema{AnyOf: anyOf}
	data, err := json.MarshalIndent(composite, "", "  ")
	if err != nil {
		return "", err
	}

	// Save the dynamically generated wrapper to disk
	if err := r.SaveGeneratedSchema(cachePath, data); err != nil {
		return "", err
	}

	return r.GetGeneratedFileURI(cachePath), nil
}
