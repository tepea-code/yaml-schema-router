# YAML Schema Router (yaml-schema-router)

[![GitHub Release](https://img.shields.io/github/v/release/traiproject/yaml-schema-router)](https://github.com/traiproject/yaml-schema-router/releases/latest)
[![CI](https://github.com/traiproject/yaml-schema-router/actions/workflows/ci.yaml/badge.svg?branch=main)](https://github.com/traiproject/yaml-schema-router/actions/workflows/ci.yaml)
[![Go Report Card](https://goreportcard.com/badge/go.trai.ch/yaml-schema-router)](https://goreportcard.com/report/go.trai.ch/yaml-schema-router)

<table width="100%">
  <tr>
    <td align="center"><b>With Router</b></td>
    <td align="center"><b>Without Router</b></td>
  </tr>
  <tr>
    <td width="50%">
      <img src="https://github.com/user-attachments/assets/df4a6614-e290-45af-a960-59e795df14e2"
           width="100%" alt="With Router" />
    </td>
    <td width="50%">
      <img src="https://github.com/user-attachments/assets/97492853-da8b-476b-a51e-8088c14024d5"
           width="100%" alt="Without Router" />
    </td>
  </tr>
</table>

## The Problem

When using the
[`yaml-language-server`](https://github.com/redhat-developer/yaml-language-server)
outside of VSCode (in editors like Helix or Neovim), getting accurate validation
for specific YAML formats is notoriously frustrating.

Most editors rely on simple file extensions or static glob patterns to assign
schemas. If you map a blanket schema to your files, you often get strict
conflicts (e.g., `"Matches multiple schemas when only one must validate"`). If
you don't map a schema, the server just checks for basic YAML syntax and ignores
type-specific validation entirely.

Existing workarounds require either strict, repetitive file-naming conventions
or manually typing modeline comments (`# yaml-language-server: $schema=...`) at
the top of every single file.

## The Solution

`yaml-schema-router` is a lightweight, editor-agnostic standard I/O (stdio)
proxy that sits between your text editor and the `yaml-language-server`. It
intercepts LSP (Language Server Protocol) traffic to dynamically route the
exact, isolated JSON schema to the language server based on the **file's content
and directory context**.

With this wrapper, you get zero-configuration, highly accurate autocomplete,
hover, and validation for supported formats in any editor, without relying on
ambiguous file extensions.

## How It Works

The wrapper acts as a transparent "Man-in-the-Middle" for the JSON-RPC
communication between the editor and the LSP (`yaml-language-server`):

1. **Traffic Interception:** The proxy listens to standard input from the
   editor. For most LSP methods, it blindly passes the JSON payload directly to
   the `yaml-language-server` with zero latency.
2. **Context Sniffing:** When the proxy detects a `textDocument/didOpen` or
   `textDocument/didChange` event, it intercepts the payload to analyze the
   file's raw text and its file path.
3. **Detector Chain:** It runs the file through a chain of "detectors" to
   identify the file type using the most reliable method for that format (e.g.,
   inspecting `apiVersion`/`kind` for K8s or directory paths for GitHub
   Workflows).
4. **Registry Lookup & Caching:** Once identified, the router checks its
   internal **Schema Registry**.
   - If the schema is already cached locally, it is served immediately
     (file-system speed).
   - If not, it is downloaded once and stored in the cache.
5. **Dynamic Schema Injection:** The proxy updates the language server's
   configuration for that specific file URI, pointing it to the locally cached
   schema.

## Key Features

- **True Editor Agnosticism:** Works with Helix, Neovim, Emacs, or any editor
  that supports standard LSP configuration.
- **Smart Detection:** Identifies files based on their actual content or
  directory location rather than just their extension.
- **Multi-Document Support:** Seamlessly handles files containing multiple YAML
  manifests (separated by `---`). The router detects all resources within the
  single file and dynamically generates a composite schema (using `anyOf`) to
  validate every manifest simultaneously.
- **Dynamic State Management:** Intelligently tracks file changes in real-time.
  If you completely empty a file, the router automatically detaches its
  associated schema. Otherwise, the existing schema remains actively cached
  until overwritten by a new recognized `apiVersion` and `kind`.
- **Local Schema Registry:** Built-in registry automatically downloads and
  caches schemas to your disk.
  - **Offline Development:** Once a schema is fetched, it is available forever,
    allowing you to work without an internet connection.
  - **High Performance:** Caching eliminates network latency on subsequent file
    opens, significantly speeding up schema injection.
- **No Modeline Clutter:** Keeps your files clean by eliminating the need for
  `# yaml-language-server: $schema=...` comments.

## Installation

You can install `yaml-schema-router` using our automated installation scripts,
manually downloading the binary, or compiling it from source via Go.

### macOS & Linux

You can easily install the latest release by running the following command in
your terminal. This script will detect your OS and architecture, download the
correct binary, and place it in `~/.local/bin` (or `/usr/local/bin` if run as
root).

```bash
curl -fsSL https://raw.githubusercontent.com/traiproject/yaml-schema-router/refs/heads/main/scripts/install.sh | sudo sh
```

or without sudo

```bash
curl -fsSL https://raw.githubusercontent.com/traiproject/yaml-schema-router/refs/heads/main/scripts/install.sh | sh
```

> [!IMPORTANT]
> If the script is ran without sudo ensure the installation directory is added
> to your system's `PATH`.

### Windows

Open **PowerShell** and run the following command to download and extract the
latest release into your user profile (`%LOCALAPPDATA%\yaml-schema-router`):

```powershell
irm https://raw.githubusercontent.com/traiproject/yaml-schema-router/refs/heads/main/scripts/install.ps1 | iex
```

**Troubleshooting "Execution of scripts is disabled":** If Windows blocks the
script from running, you need to temporarily bypass your execution policy. Run
this command first, then try the installation command again:

```powershell
Set-ExecutionPolicy Bypass -Scope Process -Force
```

### Manual Installation (All Platforms)

If you prefer not to use the automated scripts, you can download the
pre-compiled binaries directly:

1. Visit the
   [GitHub Releases page](https://github.com/traiproject/yaml-schema-router/releases/latest).
2. Download the `.tar.gz` archive for your operating system and architecture
   (e.g., `linux_x86_64`, `macOS_arm64`, `windows_x86_64`).
3. Extract the archive.
4. Move the `yaml-schema-router` executable to a directory included in your
   system's `PATH`.

### From Source (Go)

If you have Go 1.22+ installed, you can build and install the tool directly from
source:

```bash
go install go.trai.ch/yaml-schema-router/cmd/yaml-schema-router@latest
```

## Usage

Configure your editor to use `yaml-schema-router` as the language server
executable instead of `yaml-language-server`.

### Default Behavior

By default, the proxy automatically sets the following `yaml` configurations to
`true` if they are not explicitly defined in your editor's configuration:

- `hover`
- `completion`
- `validation`

> [!IMPORTANT]
> If you wish to disable any of these features, you must explicitly set them to
> `false` in your editor's LSP settings. Whatever reason you might have for
> doing so.

### Manual Schema Override (Bypassing the Router)

While `yaml-schema-router` is designed to eliminate the need for manual schema
annotations, there might be cases where you want to force a specific schema for
a single file.

If you add a standard schema modeline comment to the top of your YAML file
(within the first 10 lines), the router will automatically detect it and step
out of the way:

```yaml
# yaml-language-server: $schema=https://json.schemastore.org/github-workflow.json
name: My Custom Workflow
```

When this annotation is present, the router will stop attempting to dynamically
inject schemas for that file, allowing the underlying `yaml-language-server` to
handle the manual annotation natively. If you remove the comment later, the
router will seamlessly take over again.

### Command Line Flags

The router accepts the following flags to customize its behavior:

| Flag                   | Description                                                                                                                    | Default                                                                                                                                                                  |
| :--------------------- | :----------------------------------------------------------------------------------------------------------------------------- | :----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `--kubernetes-version` | Kubernetes schema version to use. The value must match a version directory in the upstream registry, such as `v1.36.0`.        | `v1.36.0`                                                                                                                                                                |
| `--schema-dir`         | Directory containing downloaded and user-provided schemas. Generated composite schemas are cached separately.                | The platform user cache directory under `yaml-schema-router/schemas`                                                                                                     |
| `--offline`            | Use only schemas already present in `--schema-dir`; never make schema download requests.                                      | `false`                                                                                                                                                                  |
| `--lsp-path`           | Path to the underlying `yaml-language-server` executable. Use this if the server is not in your system's PATH.                 | `yaml-language-server`                                                                                                                                                   |
| `--log-file`           | Path to a file where logs should be written. **Note:** Since the router communicates via Stdio, logs cannot be sent to stdout. | `~/.cache/yaml-schema-router/router.log` (Linux)<br>`~/Library/Caches/yaml-schema-router/router.log` (macOS)<br>`%LocalAppData%\yaml-schema-router\router.log` (Windows) |

### Example Editor Configuration (Helix)

In your `languages.toml`:

```toml
[[language]]
name = "yaml"
language-servers = [ "yaml-schema-router" ]

[language-server.yaml-schema-router]
command = "yaml-schema-router"
args = [
  "--kubernetes-version", "v1.36.0",
  "--schema-dir", "/path/to/yaml-schemas",
  "--offline",
  "--log-file", "/tmp/yaml-router.log",
  "--lsp-path", "/usr/bin/yaml-language-server"
]

# Explicitly override the proxy defaults
[language-server.yaml-schema-router.config.yaml]
hover = false
completion = false
validation = false
```

## Network & Firewall Configuration

For transparency and to assist with strict firewall or proxy rules,
`yaml-schema-router` fetches its JSON schemas from the following public
registries by default:

- **Standard Kubernetes Schemas:**
  `https://raw.githubusercontent.com/yannh/kubernetes-json-schema/master`
- **Custom Resource Definitions (CRDs):**
  `https://raw.githubusercontent.com/datreeio/CRDs-catalog/main`

The router requires outbound HTTPS (port 443) access to
`raw.githubusercontent.com` to download schemas during their first use. Because
the router utilizes a local schema registry, these network requests are only
made once per schema. Once a schema is cached locally, no further network
requests are made for that specific version, allowing for completely offline
development.

### Offline Local-Only Mode

Pass `--offline` to disable all schema downloads. The router only uses files
already stored under `--schema-dir`; it may still write generated composite and
CRD wrapper schemas to its separate platform cache.

The local directory mirrors the router's registry paths. For example:

```text
<schema-dir>/kubernetes-builtin/v1.36.0-standalone-strict/deployment-apps-v1.json
<schema-dir>/kubernetes-crd/example.com/widget_v1.json
```

When a required local schema is missing, the router writes a `WARNING` to its
configured log and leaves that resource unmapped. Other resources in the same
file whose schemas are available continue to be mapped.

## Compatibility

This tool is designed to wrap the
[**Red Hat YAML Language Server**](https://github.com/redhat-developer/yaml-language-server).

- **Recommended Version**: `1.10.0` or higher.
- **Minimum Version**: `1.0.0` (Recommended for full LSP stability).
- **Tested With**: `1.19.0+`.

While the router works with most versions that support the `yaml.schemas`
configuration setting, using a version **1.0+** ensures the best compatibility
with modern LSP clients and disables legacy behaviors (like implicit Kubernetes
schema associations) that this router is designed to replace.

### Helm Language Server (`helm-ls`)

`yaml-schema-router` can be used in conjunction with the Helm Language Server (`helm-ls`). Since `helm-ls` allows you to specify the executable for the `yaml-language-server` it delegates to, you can simply set this configuration to point to the `yaml-schema-router` binary instead.

**Current Limitation:**
Presently, there is no way to pass custom command-line flags (such as a custom `--lsp-path` for the underlying `yaml-language-server`) to `yaml-schema-router` directly through the `helm-ls` configuration. If your `yaml-language-server` is not available in your system's `PATH`, `yaml-schema-router` may fail to locate it. 

This limitation will be resolved in the future once the planned global/project configuration file support is implemented (see the [Roadmap](#roadmap)).

## Supported Detectors

### Kubernetes & CRDs

- **Standard Resources:** Automatically maps standard K8s objects (Deployments,
  Services, etc.) to the correct schema for your version.
- **CRD Support:** Automatically maps custom `apiVersion` and `kind` definitions
  to an internal Custom Resource Definition (CRD) schema registry.
  - **Schema Wrapping:** For every detected CRD, the router dynamically
    generates a **schema wrapper**. This injects standard Kubernetes
    `ObjectMeta` validation (labels, annotations, etc.) into the third-party CRD
    schema, providing a complete validation experience.

## Roadmap

- [ ] **Config File Support** (Define flags and internal defaults with a
      persistent configuration file)
