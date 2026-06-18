# managed-nvim — Step-by-Step Coding Guide

Work through these parts in order. Each part builds on the previous.
Come back to this file as you go — it will be updated as parts are completed.

---

## Part 1 — plugins.json schema
**Status: DONE**

File created: `manifest/plugins.json`
Contains all 57 plugins from your existing lazy-lock.json with full SHA pins.
This is the source of truth for every other component.

---

## Part 2 — nvim-manifest CLI
**Status: DONE**

### What you are building
A Go CLI with 5 subcommands that read and write `manifest/plugins.json`:
- `add`        — adds a plugin entry (takes explicit SHA)
- `remove`     — removes a plugin by name
- `verify`     — checks SHA format and manifest integrity
- `sign`       — CI only: signs manifest with ed25519 private key
- `verify-sig` — checks that plugins.json.sig is valid

### Step 1 — Initialize the Go module
Run from the `managed-nvim/` directory:
```bash
go mod init managed-nvim
```
Creates `go.mod`. This is Go's equivalent of package.json.
The module name `managed-nvim` must match every import path in your code.

### Step 2 — Create the directory structure
```bash
mkdir -p cmd/nvim-manifest
mkdir -p internal/manifest
```
- `cmd/` — one folder per binary you will build
- `internal/` — shared code that Go prevents anything outside this module from importing

### Step 3 — internal/manifest/manifest.go
This file defines the shared types every binary uses.
Create `internal/manifest/manifest.go`:

```go
package manifest

import (
    "encoding/json"
    "fmt"
    "os"
    "time"
)

type Plugin struct {
    Name       string `json:"name"`
    Repo       string `json:"repo"`
    Upstream   string `json:"upstream"`
    Branch     string `json:"branch"`
    SHA        string `json:"sha"`
    ApprovedAt string `json:"approved_at"`
    ApprovedBy string `json:"approved_by"`
}

type Manifest struct {
    SchemaVersion int      `json:"schema_version"`
    LastUpdated   string   `json:"last_updated"`
    Plugins       []Plugin `json:"plugins"`
}

func Load(path string) (*Manifest, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("reading manifest: %w", err)
    }
    var m Manifest
    if err := json.Unmarshal(data, &m); err != nil {
        return nil, fmt.Errorf("parsing manifest: %w", err)
    }
    return &m, nil
}

func (m *Manifest) Save(path string) error {
    m.LastUpdated = time.Now().Format("2006-01-02")
    data, err := json.MarshalIndent(m, "", "  ")
    if err != nil {
        return fmt.Errorf("encoding manifest: %w", err)
    }
    return os.WriteFile(path, append(data, '\n'), 0644)
}

func (m *Manifest) FindByName(name string) *Plugin {
    for i := range m.Plugins {
        if m.Plugins[i].Name == name {
            return &m.Plugins[i]
        }
    }
    return nil
}
```

**Go concepts:**
- Struct tags like `json:"name"` tell encoding/json which JSON key maps to which field
- `(m *Manifest)` on Save is a method receiver — Save belongs to Manifest. The `*` means pointer (pass by reference)
- Functions return `(result, error)` — Go has no exceptions. Always check `if err != nil`
- `fmt.Errorf("context: %w", err)` wraps errors with context. `%w` lets callers unwrap later
- `time.Now().Format("2006-01-02")` — Go uses a reference date instead of Y-m-d codes

### Step 4 — cmd/nvim-manifest/main.go
Create `cmd/nvim-manifest/main.go`:

```go
package main

import (
    "fmt"
    "os"
)

func main() {
    if len(os.Args) < 2 {
        printUsage()
        os.Exit(1)
    }

    manifestPath := os.Getenv("MANIFEST_PATH")
    if manifestPath == "" {
        manifestPath = "manifest/plugins.json"
    }

    cmd := os.Args[1]
    args := os.Args[2:]

    var err error
    switch cmd {
    case "add":
        err = runAdd(manifestPath, args)
    case "remove":
        err = runRemove(manifestPath, args)
    case "verify":
        err = runVerify(manifestPath)
    case "sign":
        err = runSign(manifestPath, args)
    case "verify-sig":
        err = runVerifySig(manifestPath, args)
    default:
        fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
        printUsage()
        os.Exit(1)
    }

    if err != nil {
        fmt.Fprintf(os.Stderr, "error: %v\n", err)
        os.Exit(1)
    }
}

func printUsage() {
    fmt.Println(`nvim-manifest — managed-nvim plugin manifest tool

Usage:
  nvim-manifest add <owner/repo> <sha> <approved-by>
  nvim-manifest remove <plugin-name>
  nvim-manifest verify
  nvim-manifest sign <private-key-file>
  nvim-manifest verify-sig <public-key-file>`)
}
```

**Go concepts:**
- `os.Args` is the slice of CLI arguments. os.Args[0] = binary name, os.Args[1] = first arg
- `os.Getenv` reads an environment variable, returns "" if not set
- `fmt.Fprintf(os.Stderr, ...)` prints to stderr — errors go here so scripts can separate them from output
- Go's switch does not fall through by default (unlike PHP/JS)

### Step 5 — cmd/nvim-manifest/add.go
`add` takes an explicit SHA — no network call needed.
The SHA comes from whoever is requesting the plugin (they look it up on GitHub).
Once Part 3 (nvim-mirror) exists, this will be updated to query Artifactory instead.

Create `cmd/nvim-manifest/add.go`:

```go
package main

import (
    "fmt"
    "strings"
    "time"

    "managed-nvim/internal/manifest"
)

func runAdd(manifestPath string, args []string) error {
    if len(args) < 3 {
        return fmt.Errorf("usage: nvim-manifest add <owner/repo> <sha> <approved-by>")
    }

    repo := args[0]
    sha := args[1]
    approvedBy := args[2]

    parts := strings.SplitN(repo, "/", 2)
    if len(parts) != 2 {
        return fmt.Errorf("repo must be owner/name format, got: %s", repo)
    }
    name := parts[1]

    if len(sha) != 40 {
        return fmt.Errorf("sha must be a 40-character git commit hash, got: %s", sha)
    }

    m, err := manifest.Load(manifestPath)
    if err != nil {
        return err
    }

    if existing := m.FindByName(name); existing != nil {
        return fmt.Errorf("plugin %q already exists (sha: %s)", name, existing.SHA)
    }

    m.Plugins = append(m.Plugins, manifest.Plugin{
        Name:       name,
        Repo:       repo,
        Upstream:   "https://github.com/" + repo,
        Branch:     "main",
        SHA:        sha,
        ApprovedAt: time.Now().Format("2006-01-02"),
        ApprovedBy: approvedBy,
    })

    if err := m.Save(manifestPath); err != nil {
        return err
    }

    fmt.Printf("added %s @ %s\n", repo, sha[:8])
    return nil
}
```

**Note:** Branch defaults to "main". You can add an optional `--branch` flag later.

### Step 6 — cmd/nvim-manifest/remove.go
Create `cmd/nvim-manifest/remove.go`:

```go
package main

import (
    "fmt"

    "managed-nvim/internal/manifest"
)

func runRemove(manifestPath string, args []string) error {
    if len(args) < 1 {
        return fmt.Errorf("usage: nvim-manifest remove <plugin-name>")
    }
    name := args[0]

    m, err := manifest.Load(manifestPath)
    if err != nil {
        return err
    }

    before := len(m.Plugins)
    filtered := m.Plugins[:0]
    for _, p := range m.Plugins {
        if p.Name != name {
            filtered = append(filtered, p)
        }
    }
    m.Plugins = filtered

    if len(m.Plugins) == before {
        return fmt.Errorf("plugin %q not found", name)
    }

    if err := m.Save(manifestPath); err != nil {
        return err
    }

    fmt.Printf("removed %s\n", name)
    return nil
}
```

**Go concept — slice filtering:**
`m.Plugins[:0]` creates a zero-length slice sharing the same underlying array.
The loop keeps only items you want. No new allocation needed. Common Go pattern.

### Step 7 — cmd/nvim-manifest/verify.go
For now, verify checks that every SHA is exactly 40 hex characters.
Once Part 3 (nvim-mirror) exists, update this to also confirm each plugin
archive exists in Artifactory.

Create `cmd/nvim-manifest/verify.go`:

```go
package main

import (
    "fmt"
    "regexp"

    "managed-nvim/internal/manifest"
)

var shaPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

func runVerify(manifestPath string) error {
    m, err := manifest.Load(manifestPath)
    if err != nil {
        return err
    }

    failed := 0
    for _, p := range m.Plugins {
        if !shaPattern.MatchString(p.SHA) {
            fmt.Printf("FAIL  %s — invalid SHA: %q\n", p.Name, p.SHA)
            failed++
            continue
        }
        if p.Repo == "" || p.Upstream == "" {
            fmt.Printf("FAIL  %s — missing repo or upstream\n", p.Name)
            failed++
            continue
        }
        fmt.Printf("OK    %s @ %s\n", p.Name, p.SHA[:8])
    }

    fmt.Printf("\n%d plugins, %d failed\n", len(m.Plugins), failed)
    if failed > 0 {
        return fmt.Errorf("%d plugin(s) failed verification", failed)
    }
    return nil
}
```

### Step 8 — cmd/nvim-manifest/sign.go
Signs `plugins.json` using ed25519. Only used in CI — the private key never
lives on a developer machine.

Create `cmd/nvim-manifest/sign.go`:

```go
package main

import (
    "crypto/ed25519"
    "encoding/pem"
    "fmt"
    "os"

    "managed-nvim/internal/manifest"
)

func runSign(manifestPath string, args []string) error {
    if len(args) < 1 {
        return fmt.Errorf("usage: nvim-manifest sign <private-key-file>")
    }
    keyPath := args[0]

    // Read and parse the private key
    keyPEM, err := os.ReadFile(keyPath)
    if err != nil {
        return fmt.Errorf("reading private key: %w", err)
    }
    block, _ := pem.Decode(keyPEM)
    if block == nil {
        return fmt.Errorf("private key file is not valid PEM")
    }
    if len(block.Bytes) != ed25519.PrivateKeySize {
        return fmt.Errorf("unexpected key size: %d bytes", len(block.Bytes))
    }
    privateKey := ed25519.PrivateKey(block.Bytes)

    // Read the raw manifest bytes (sign exactly what is on disk)
    data, err := os.ReadFile(manifestPath)
    if err != nil {
        return fmt.Errorf("reading manifest: %w", err)
    }

    // Validate it parses correctly before signing
    var m manifest.Manifest
    if err := validateJSON(data, &m); err != nil {
        return fmt.Errorf("manifest is not valid JSON: %w", err)
    }

    sig := ed25519.Sign(privateKey, data)

    sigPath := manifestPath + ".sig"
    if err := os.WriteFile(sigPath, sig, 0644); err != nil {
        return fmt.Errorf("writing signature: %w", err)
    }

    fmt.Printf("signed %s → %s\n", manifestPath, sigPath)
    fmt.Printf("key:  %x\n", privateKey.Public())
    fmt.Printf("sig:  %x\n", sig[:8])
    return nil
}
```

You also need a small helper. Add to the bottom of `sign.go`:

```go
import "encoding/json"

func validateJSON(data []byte, v any) error {
    return json.Unmarshal(data, v)
}
```

**Generating a key pair (run once, keep private key in CI secrets only):**
```bash
# You will write a small keygen helper or use openssl:
openssl genpkey -algorithm ed25519 -out signing.key
openssl pkey -in signing.key -pubout -out signing.pub
```

### Step 9 — cmd/nvim-manifest/verify_sig.go
Verifies that `plugins.json.sig` was produced by the matching private key.
Used for local testing. The wrapper binary does the same check at startup
but uses the public key compiled in at build time.

Create `cmd/nvim-manifest/verify_sig.go`:

```go
package main

import (
    "crypto/ed25519"
    "encoding/pem"
    "fmt"
    "os"
)

func runVerifySig(manifestPath string, args []string) error {
    if len(args) < 1 {
        return fmt.Errorf("usage: nvim-manifest verify-sig <public-key-file>")
    }
    keyPath := args[0]

    // Read public key
    keyPEM, err := os.ReadFile(keyPath)
    if err != nil {
        return fmt.Errorf("reading public key: %w", err)
    }
    block, _ := pem.Decode(keyPEM)
    if block == nil {
        return fmt.Errorf("public key file is not valid PEM")
    }
    publicKey := ed25519.PublicKey(block.Bytes)

    // Read manifest
    data, err := os.ReadFile(manifestPath)
    if err != nil {
        return fmt.Errorf("reading manifest: %w", err)
    }

    // Read signature
    sig, err := os.ReadFile(manifestPath + ".sig")
    if err != nil {
        return fmt.Errorf("reading signature: %w", err)
    }

    if !ed25519.Verify(publicKey, data, sig) {
        return fmt.Errorf("signature verification FAILED — manifest may have been tampered with")
    }

    fmt.Println("signature OK")
    return nil
}
```

### Step 10 — Build and test
```bash
# Build
go build ./cmd/nvim-manifest/

# Verify your 57 plugins
./nvim-manifest verify

# Add a test plugin (use a real 40-char SHA from any GitHub commit)
./nvim-manifest add folke/zen-mode.nvim a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2 yourname

# Remove it
./nvim-manifest remove zen-mode.nvim

# Verify again — should still be 57 plugins, all OK
./nvim-manifest verify
```

---

## Part 3 — nvim-mirror CLI
**Status: DONE**

Reads `manifest/plugins.json` and mirrors every plugin into Artifactory
as a `.tar.gz` archive. Only runs in CI.

### What you will build
`cmd/nvim-mirror/main.go` — single binary, no subcommands needed.

### Directory structure to create
```bash
mkdir -p cmd/nvim-mirror
```

### What it does (per plugin)
1. `git clone --depth 1 --branch <branch> <upstream>` into a temp dir
2. `git checkout <sha>` to pin to the exact commit
3. Calls `nvim-scan` against the cloned source — aborts if scan fails
4. `tar.gz` the directory
5. HTTP PUT to Artifactory:
   `https://artifactory.company.com/nvim-plugins/<repo>/<sha>.tar.gz`
6. Prints result per plugin, writes a JSON run report

### Artifactory path convention
```
nvim-plugins/folke/lazy.nvim/306a055.tar.gz
nvim-plugins/nvim-treesitter/nvim-treesitter/7c14161.tar.gz
```

### Go concepts you will learn
- `os/exec` — running git as a subprocess
- `archive/tar` + `compress/gzip` — creating .tar.gz archives
- `net/http` — HTTP PUT to Artifactory REST API
- `io` — streaming data without loading it all into memory
- `os.MkdirTemp` — creating and cleaning up temp directories

### Environment variables it reads
```
ARTIFACTORY_URL      https://artifactory.company.com
ARTIFACTORY_REPO     nvim-plugins
ARTIFACTORY_USER     ci-user
ARTIFACTORY_TOKEN    <api-token>
MANIFEST_PATH        manifest/plugins.json  (default)
```

---

## Part 4 — nvim-scan CLI
**Status: IN PROGRESS**

Static analysis of plugin Lua source. Runs inside the mirror pipeline.
A plugin that fails scan never reaches Artifactory.

### What you will build
`cmd/nvim-scan/main.go`

### Usage
```bash
nvim-scan <directory>
# exits 0 = clean, exits 1 = findings, prints JSON report to stdout
```

### Checks to implement
| Check | Pattern | Severity |
|---|---|---|
| Hardcoded URL | `https?://` outside a comment | WARN |
| Shell execution | `io.popen`, `os.execute` | ERROR |
| Neovim job with args | `vim.fn.jobstart`, `vim.fn.system` | WARN |
| Obfuscated execution | `load(` with string arg | ERROR |
| AI config reference | `.claude`, `.copilot`, `.vscode` | ERROR |
| SSH config reference | `.ssh/` | ERROR |
| Blocklist match | strings from `manifest/blocklist.json` | ERROR |

### Go concepts you will learn
- `path/filepath.Walk` — recursively walking a directory
- `regexp` — compiling and matching patterns
- `bufio.Scanner` — reading files line by line efficiently

---

## Part 5-7 — nvim-wrapper
**Status: TODO — most important binary, build after Parts 3 and 4**

Replaces the `nvim` binary on every developer machine.
See README.md for the full startup/exit sequence.

### Files to create
```
cmd/nvim-wrapper/main.go           — startup sequence, shared logic
cmd/nvim-wrapper/platform_darwin.go — chflags + sandbox-exec
cmd/nvim-wrapper/platform_linux.go  — chattr + Landlock
```

### Build tags
Each platform file starts with:
```go
//go:build darwin   (platform_darwin.go)
//go:build linux    (platform_linux.go)
```
Go only compiles the file matching the target OS.

### Public key injection at build time
```bash
go build \
  -ldflags="-X main.manifestPublicKey=$(base64 -i signing.pub)" \
  ./cmd/nvim-wrapper/
```
The public key is then a constant inside the binary —
not a file, not an env var, not readable or replaceable by a normal user.

### Go concepts you will learn
- `syscall` / `golang.org/x/sys/unix` — OS-level calls (ioctl, Landlock)
- `syscall.Exec` — replacing the current process with nvim (not a subprocess)
- `crypto/sha256` — hashing files for integrity checks
- `defer` — cleanup that runs when the function returns (used for exit trap)
- Build tags — conditional compilation per OS

---

## Part 8-11 — Neovim Lua config
**Status: TODO — build after wrapper so you can test end to end**

### Files to create
```
nvim/init.lua
nvim/lua/managed/bootstrap.lua
nvim/lua/managed/whitelist.lua
nvim/lua/managed/network_shim.lua
nvim/lua/managed/hooks_blocker.lua
nvim/lua/managed/audit.lua
```

### Load order (important)
```
init.lua
  └─ managed/bootstrap.lua    loads first — sets up lazy.nvim
  └─ managed/whitelist.lua    runs before any plugin loads
  └─ managed/network_shim.lua wraps jobstart/system
  └─ managed/hooks_blocker.lua clears banned autocmds
  └─ managed/audit.lua        sets up logging
  └─ plugins/*.lua            your approved plugin specs
  └─ plugin/managed_lock.lua  loads LAST — final enforcement check
```

---

## Part 12 — nvim-hook-sanitizer
**Status: TODO**

Strips unapproved hooks from AI tool config files before Neovim starts.
Runs as part of the wrapper startup sequence.

### Files it sanitizes
- `~/.claude/settings.json` — removes hooks with unapproved commands
- `~/.vscode/settings.json` — removes unapproved tasks
- `~/.vscode/tasks.json`    — removes unapproved task definitions

### Allowlist
Maintained in `manifest/hook-allowlist.json`:
```json
{
  "allowed_commands": ["prettier", "eslint", "go fmt", "gofmt", "black"]
}
```

---

## Part 13 — install.sh
**Status: TODO**

POSIX shell script. Works on macOS and all Linux distros.

### What it does
1. Downloads the managed-nvim release tarball from Artifactory
2. Verifies the tarball checksum
3. Extracts to `/opt/managed-nvim/`
4. Symlinks `/usr/local/bin/nvim` → `/opt/managed-nvim/bin/nvim-wrapper`
5. Sets up the background updater cron entry
6. Prints one-line success message

---

## Part 14 — deb + rpm packaging
**Status: TODO — build after install.sh**

Uses `fpm` to wrap the install into proper packages for fleet management.

```bash
# deb
fpm -s dir -t deb -n managed-nvim -v 1.0.0 /opt/managed-nvim

# rpm
fpm -s dir -t rpm -n managed-nvim -v 1.0.0 /opt/managed-nvim
```

---

## Part 15 — CI pipeline
**Status: TODO — last, wires everything together**

Runs on a schedule (nightly) and on every merge to the manifest repo.

```
nvim-manifest verify
  → nvim-mirror (clones + nvim-scan + pushes to Artifactory)
  → nvim-manifest sign /secrets/signing.key
  → publish plugins.json + plugins.json.sig to Artifactory
  → build nvim-wrapper with public key injected
  → build install.sh release tarball
  → publish release to Artifactory
```
