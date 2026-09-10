// Package kubernetes implements a schema detector for standard Kubernetes manifests.
package kubernetes

import (
	"fmt"
	"log"
	"net/url"
	"path/filepath"
	"strings"

	"go.trai.ch/yaml-schema-router/internal/config"
	"go.trai.ch/yaml-schema-router/internal/detector"
	"go.trai.ch/yaml-schema-router/internal/schemaregistry"
)

// K8sDetector implements the detector.Detector interface for Kubernetes manifests.
type K8sDetector struct {
	Registry *schemaregistry.Registry
	Version  string
}

var _ detector.Detector = (*K8sDetector)(nil)

// K8sDetectorName is the unique identifier for the built-in Kubernetes detector.
const K8sDetectorName = "kubernetes-builtin"

// Name returns the unique string identifier for the Kubernetes detector.
func (d *K8sDetector) Name() string {
	return K8sDetectorName
}

type typeMeta struct {
	APIVersion string
	Kind       string
}

// Detect inspects the YAML content for all Kubernetes apiVersion and kind pairs
// to construct the appropriate schema URLs.
func (d *K8sDetector) Detect(_ string, content []byte) ([]string, error) {
	metas := extractAllTypeMeta(content)
	if len(metas) == 0 {
		return nil, nil
	}

	var schemaURLs []string

	for _, meta := range metas {
		if schemaURL := d.resolveSchemaURL(meta); schemaURL != "" {
			schemaURLs = append(schemaURLs, schemaURL)
		}
	}

	return schemaURLs, nil
}

func (d *K8sDetector) resolveSchemaURL(meta typeMeta) string {
	log.Printf("[%s] Found apiVersion='%s', kind='%s'", d.Name(), meta.APIVersion, meta.Kind)

	if meta.Kind == "CustomResourceDefinition" {
		log.Printf("[%s] Ignoring CustomResourceDefinition", d.Name())
		return ""
	}

	group := meta.APIVersion
	version := ""
	if strings.Contains(group, "/") {
		parts := strings.Split(group, "/")
		group = parts[0]
		version = parts[1]
	}

	// If the group contains a dot but doesn't end with k8s.io, it is a custom resource.
	if strings.Contains(group, ".") && !strings.HasSuffix(group, "k8s.io") {
		log.Printf("[%s] Ignoring Custom Resource (group: %s)", d.Name(), group)
		return ""
	}

	// Standardize the API group name for the schema registry by stripping the domain
	// e.g., "rbac.authorization.k8s.io" -> "rbac", "networking.k8s.io" -> "networking"
	formattedGroup := group
	if strings.Contains(formattedGroup, ".") && strings.HasSuffix(formattedGroup, "k8s.io") {
		formattedGroup = strings.Split(formattedGroup, ".")[0]
	}

	var apiVersionFormatted string
	if version != "" {
		apiVersionFormatted = fmt.Sprintf("%s-%s", formattedGroup, version)
	} else {
		apiVersionFormatted = formattedGroup // For core groups like "v1"
	}

	kindFormatted := strings.ToLower(meta.Kind)
	fileName := fmt.Sprintf("%s-%s.json", kindFormatted, apiVersionFormatted)
	versionDir := fmt.Sprintf("%s%s", d.Version, config.DefaultK8sSchemaFlavour)

	remoteSchemaURL, err := url.JoinPath(
		config.DefaultK8sSchemaRegistry,
		versionDir,
		fileName,
	)
	if err != nil {
		log.Printf("[%s] Failed to build URL for %s: %v", d.Name(), meta.Kind, err)
		return ""
	}

	cachePath := filepath.Join(d.Name(), versionDir, fileName)
	localURI, err := d.Registry.GetSchemaURI(remoteSchemaURL, cachePath)
	if err != nil {
		log.Printf("[%s] Failed to fetch schema for %s: %v", d.Name(), meta.Kind, err)
		return ""
	}

	return localURI
}

// extractAllTypeMeta splits the raw YAML content by document separators
// and extracts the apiVersion and kind for each segment.
func extractAllTypeMeta(content []byte) []typeMeta {
	var metas []typeMeta
	docs := strings.SplitSeq(string(content), "---")

	for doc := range docs {
		var apiVersion, kind string
		for line := range strings.SplitSeq(doc, "\n") {
			// Only check top-level keys
			if after, ok := strings.CutPrefix(line, "apiVersion:"); ok {
				apiVersion = strings.TrimSpace(after)
				apiVersion = strings.Trim(apiVersion, `"'`)
			} else if after0, ok0 := strings.CutPrefix(line, "kind:"); ok0 {
				kind = strings.TrimSpace(after0)
				kind = strings.Trim(kind, `"'`)
			}

			if apiVersion != "" && kind != "" {
				break
			}
		}

		if apiVersion != "" && kind != "" {
			metas = append(metas, typeMeta{APIVersion: apiVersion, Kind: kind})
		}
	}

	return metas
}
