// Package config holds the application-wide configuration, default settings,
// and file permission constants for k8s-yaml-router.
package config

import (
	"os"
	"time"
)

const (
	// DefaultDirPerm represents standard directory permissions (rwxr-xr-x).
	DefaultDirPerm os.FileMode = 0o755

	// DefaultFilePerm represents standard file permissions (rw-rw-rw-).
	DefaultFilePerm os.FileMode = 0o666

	// DefaultConfigDirName is the folder name inside ~/.config/.
	DefaultConfigDirName = "yaml-schema-router"

	// DefaultK8sSchemaRegistry is the url to fetch k8s schmeas from.
	DefaultK8sSchemaRegistry = "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/master"

	// DefaultK8sSchemaVersion is the version of the k8s schmeas to fetch.
	DefaultK8sSchemaVersion = "v1.36.0"

	// DefaultK8sSchemaFlavour is the "-standalone-strict" suffix for self-contained, strict validation.
	DefaultK8sSchemaFlavour = "-standalone-strict"

	// DefaultCRDSchemaRegistry is the url to fetch crd schmas from.
	DefaultCRDSchemaRegistry = "https://raw.githubusercontent.com/datreeio/CRDs-catalog/main"

	// DefaultK8sMetaSchemaFileName is the filename of the Kubernetes ObjectMeta schema.
	DefaultK8sMetaSchemaFileName = "objectmeta-meta-v1.json"

	// DefaultHover determines if hover support is enabled by default.
	DefaultHover = true

	// DefaultCompletion determines if completion support is enabled by default.
	DefaultCompletion = true

	// DefaultValidation determines if validation support is enabled by default.
	DefaultValidation = true

	// DefaultDownloaderTimeout is the maximum duration allowed for schema downloads.
	DefaultDownloaderTimeout = 2 * time.Second
)
