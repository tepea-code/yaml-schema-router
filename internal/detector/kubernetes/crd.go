package kubernetes

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"go.trai.ch/yaml-schema-router/internal/config"
	"go.trai.ch/yaml-schema-router/internal/detector"
	"go.trai.ch/yaml-schema-router/internal/schemaregistry"
)

type schemaRef struct {
	Ref string `json:"$ref"`
}

type schemaProperties struct {
	Metadata schemaRef `json:"metadata"`
}

type schemaExtension struct {
	Properties schemaProperties `json:"properties"`
}

type schemaWrapper struct {
	AllOf []any `json:"allOf"`
}

// CRDDetector implements the detector.Detector interface for Kubernetes CRDs.
type CRDDetector struct {
	Registry *schemaregistry.Registry
	Version  string
}

var _ detector.Detector = (*CRDDetector)(nil)

// CRDDetectorName is the unique identifier for the built-in Kubernetes detector.
const CRDDetectorName = "kubernetes-crd"

// Name returns the unique string identifier for the CRD detector.
func (d *CRDDetector) Name() string {
	return CRDDetectorName
}

// Detect inspects the YAML content for apiVersions containing custom groups
// and constructs wrapped JSON schemas that include standard ObjectMeta.
func (d *CRDDetector) Detect(_ string, content []byte) ([]string, error) {
	metas := extractAllTypeMeta(content)
	if len(metas) == 0 {
		return nil, nil
	}

	schemaURLs := make([]string, 0, len(metas))

	for _, meta := range metas {
		group, version, found := strings.Cut(meta.APIVersion, "/")
		if !found || (!strings.Contains(group, ".") || strings.HasSuffix(group, "k8s.io")) {
			continue // Not a CRD, let the builtin detector handle it
		}

		log.Printf("[%s] Detected Custom Resource: %s/%s", d.Name(), group, meta.Kind)

		kindFormatted := strings.ToLower(meta.Kind)
		fileName := fmt.Sprintf("%s_%s.json", kindFormatted, version)
		localBaseCRDURI, localObjectMetaURI, err := d.fetchDependencies(group, fileName)
		if err != nil {
			log.Printf("[%s] Failed to fetch dependencies for CRD %s: %v", d.Name(), meta.Kind, err)
			continue
		}

		wrapperID := sha256.Sum256([]byte(localBaseCRDURI + "\x00" + localObjectMetaURI))
		wrapperCachePath := filepath.Join(
			CRDDetectorName,
			group,
			fmt.Sprintf("%s_%s_wrapper_%x.json", kindFormatted, version, wrapperID[:8]),
		)

		if _, statErr := os.Stat(d.Registry.GetGeneratedPath(wrapperCachePath)); statErr == nil {
			log.Printf("[%s] Wrapper cache hit for %s", d.Name(), wrapperCachePath)
			schemaURLs = append(schemaURLs, d.Registry.GetGeneratedFileURI(wrapperCachePath))
			continue
		}

		// Generate and save the wrapper schema
		fileURI, err := d.generateAndSaveWrapper(localBaseCRDURI, localObjectMetaURI, wrapperCachePath)
		if err != nil {
			log.Printf("[%s] Failed to generate wrapper for CRD %s: %v", d.Name(), meta.Kind, err)
			continue
		}

		schemaURLs = append(schemaURLs, fileURI)
	}

	return schemaURLs, nil
}

func (d *CRDDetector) fetchDependencies(
	group, fileName string,
) (localBaseCRDURI, localObjectMetaURI string, err error) {
	// Get base CRD remote URL & fetch local URI
	baseCRDURL, err := url.JoinPath(
		config.DefaultCRDSchemaRegistry,
		group,
		fileName,
	)
	if err != nil {
		return "", "", err
	}
	baseCRDCachePath := filepath.Join(d.Name(), group, fileName)
	localBaseCRDURI, err = d.Registry.GetSchemaURI(baseCRDURL, baseCRDCachePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch base CRD schema: %w", err)
	}

	// Get ObjectMeta remote URL & fetch local URI
	versionDir := fmt.Sprintf("%s%s", d.Version, config.DefaultK8sSchemaFlavour)
	objectMetaURL, err := url.JoinPath(config.DefaultK8sSchemaRegistry, versionDir, config.DefaultK8sMetaSchemaFileName)
	if err != nil {
		return "", "", err
	}
	metaCachePath := filepath.Join(K8sDetectorName, versionDir, config.DefaultK8sMetaSchemaFileName)
	localObjectMetaURI, err = d.Registry.GetSchemaURI(objectMetaURL, metaCachePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch ObjectMeta schema: %w", err)
	}

	return localBaseCRDURI, localObjectMetaURI, nil
}

// generateAndSaveWrapper builds the CRD wrapper and saves it to the persistent cache.
func (d *CRDDetector) generateAndSaveWrapper(
	localBaseCRDURI, localObjectMetaURI, wrapperCachePath string,
) (string, error) {
	log.Printf("[%s] Generating schema wrapper: %s + %s -> %s",
		d.Name(),
		localBaseCRDURI,
		localObjectMetaURI,
		wrapperCachePath,
	)

	wrapper := schemaWrapper{
		AllOf: []any{
			schemaRef{Ref: localBaseCRDURI},
			schemaExtension{
				Properties: schemaProperties{
					Metadata: schemaRef{Ref: localObjectMetaURI},
				},
			},
		},
	}

	wrapperBytes, err := json.MarshalIndent(wrapper, "", "  ")
	if err != nil {
		return "", err
	}

	// Write the generated schema to the persistent cache directory
	if err := d.Registry.SaveGeneratedSchema(wrapperCachePath, wrapperBytes); err != nil {
		return "", err
	}

	return d.Registry.GetGeneratedFileURI(wrapperCachePath), nil
}
