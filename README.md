# managed-nvim

A managed, locked-down Neovim distribution for corporate fleets.
Plugins are whitelisted, mirrored through JFrog Artifactory, and protected
against supply-chain attacks (Miasma / Shai-Halud threat model).

---

## Threat Model

Target attack: a whitelisted plugin receives a malicious update that:
- Writes hooks into `.claude/settings.json`, `.copilot/`, `.vscode/settings.json`
- Uses those hooks to exfiltrate code or tokens on the next AI tool event
- Uses Neovim autocmds / jobstart to phone home

Defense principle: **even a fully compromised whitelisted plugin must not be
able to do any of this.** Enforcement is at the OS level, below Lua.

---

## Key Decisions

| Decision | Choice | Reason |
|---|---|---|
| Language | Go | Single static binary, trivial cross-compilation, good syscall support |
| Plugin manager | lazy.nvim (unforked) | Thin Lua shim overrides setup(); fork has too much maintenance cost |
| Plugin source | Generic package registry (Gitea, Artifactory, Nexus, etc.) as `.tar.gz` | Installer fetches archives; Neovim itself never touches the network for plugins |
| Manifest source | Central Git repo → signed → published to Artifactory | PR = approval workflow; signing prevents local substitution |
| Signing algorithm | ed25519 (Go stdlib crypto/ed25519) | Fast, small keys, no external deps |
| Lock level | Hard — non-whitelisted plugin = Lua error at startup | |
| Monitoring | Plugin source scan (CI) + runtime audit log + pluggable HTTP shipper | |
| Distribution | Shell install script + deb + rpm | No Nix, no Homebrew |
| OS targets | macOS, Ubuntu/Debian, RHEL/CentOS | |

---

## How It Works (Employee Perspective)

**Install (once):**
```bash
curl https://artifactory.company.com/managed-nvim/install.sh | sh
```
Installs to `/opt/managed-nvim/`, puts `nvim` wrapper in PATH. ~30 seconds, zero config.

**Daily use:**
Employee types `nvim`. The wrapper runs invisibly (<100ms overhead):
1. Fetches latest signed manifest from Artifactory (cached, only re-fetches when stale)
2. Verifies manifest signature against public key compiled into the binary
3. Hashes `.claude/`, `.copilot/`, `.vscode/`, `.ssh/`, `.gitconfig`
4. Sets those files immutable (chattr +i / chflags uchg)
5. Launches Neovim inside an OS sandbox (Landlock on Linux, sandbox-exec on macOS)
6. On exit: removes immutability, compares hashes, ships audit log

**Plugin updates:**
A background updater runs on a schedule (not on nvim startup).
Employee sees a one-line note in their next shell session:
```
managed-nvim: updated 3 plugins (neogit, blink.cmp, telescope)
```

**Wanting an unapproved plugin:**
Neovim shows:
```
[managed-nvim] BLOCKED: 'folke/zen-mode.nvim' is not whitelisted.
Request approval: https://your-wiki/nvim-plugin-request
```

---

## Repository Layout

```
managed-nvim/
├── manifest/
│   ├── plugins.json          # approved plugin list (source of truth)
│   └── plugins.json.sig      # ed25519 signature — produced by CI
│
├── internal/
│   └── manifest/
│       └── manifest.go       # shared Go types: Plugin, Manifest, Load, Save
│
├── cmd/
│   ├── nvim-manifest/        # manifest management CLI
│   │   ├── main.go
│   │   ├── add.go
│   │   ├── remove.go
│   │   ├── verify.go         # checks all SHAs exist upstream (Artifactory)
│   │   ├── sign.go           # CI only: signs plugins.json → plugins.json.sig
│   │   └── verify_sig.go     # checks the signature is valid
│   │
│   ├── nvim-mirror/          # CI tool: mirrors plugins GitHub → Artifactory
│   │   └── main.go
│   │
│   ├── nvim-scan/            # CI tool: static analysis of plugin Lua source
│   │   └── main.go
│   │
│   ├── nvim-wrapper/         # replaces the nvim binary on developer machines
│   │   ├── main.go
│   │   ├── platform_darwin.go
│   │   └── platform_linux.go
│   │
│   └── nvim-hook-sanitizer/  # strips unapproved hooks from AI tool configs
│       └── main.go
│
├── nvim/                     # the managed Neovim Lua config
│   ├── init.lua
│   └── lua/
│       └── managed/
│           ├── bootstrap.lua     # lazy.nvim setup with Artifactory shim
│           ├── whitelist.lua     # hard-errors on unapproved plugins
│           ├── network_shim.lua  # wraps jobstart/system (defense in depth)
│           ├── hooks_blocker.lua # clears banned autocmd patterns
│           └── audit.lua         # runtime audit log + HTTP shipping
│
└── scripts/
    ├── install.sh            # employee installer
    ├── build-deb.sh
    └── build-rpm.sh
```

---

## Build Plan

### Status

| Part | What | Status |
|---|---|---|
| 1 | `manifest/plugins.json` schema | DONE |
| 2 | `nvim-manifest` CLI (add, remove, verify, sign, verify-sig) | DONE |
| 3 | `nvim-mirror` CLI (Gitea generic package registry) | DONE |
| 4 | `nvim-scan` CLI | SKIPPED |
| 5 | `nvim-wrapper` core | DONE |
| 6 | `nvim-wrapper` macOS (chflags + sandbox-exec) | DONE |
| 7 | `nvim-wrapper` Linux (chattr + Landlock) | STUB |
| 8 | `nvim/init.lua` skeleton + load order | DONE |
| 9 | `managed/whitelist.lua` | DONE |
| 10 | `managed/network_shim.lua` | TODO |
| 11 | `managed/hooks_blocker.lua` + `managed/audit.lua` | TODO |
| 12 | `nvim-hook-sanitizer` CLI | TODO |
| 13 | `scripts/install.sh` | TODO |
| 14 | deb + rpm packaging | TODO |
| 15 | CI pipeline definition | TODO |

---

## Part 2 — nvim-manifest CLI (detail)

**Files:**
```
cmd/nvim-manifest/
  main.go         — subcommand routing, reads MANIFEST_PATH env var
  add.go          — adds a plugin (takes explicit SHA, no network call)
  remove.go       — removes a plugin by name
  verify.go       — checks all plugin SHAs exist in Artifactory
  sign.go         — CI only: signs manifest with ed25519 private key
  verify_sig.go   — verifies plugins.json.sig against compiled-in public key
```

**Usage:**
```bash
nvim-manifest add <owner/repo> <sha> <approved-by>
nvim-manifest remove <plugin-name>
nvim-manifest verify
nvim-manifest sign <path-to-private-key>     # CI only
nvim-manifest verify-sig
```

**Notes:**
- `add` takes an explicit SHA rather than fetching from GitHub/Artifactory.
  Once Part 3 (nvim-mirror) exists, revisit this to query Artifactory instead.
- `verify` will check Artifactory once Part 3 exists. For now it can check
  that the SHA format is valid (64 hex chars).
- `sign` and `verify-sig` use ed25519 from Go's crypto/ed25519 stdlib.
- The public key is compiled into nvim-wrapper via -ldflags at build time.
  nvim-manifest verify-sig uses the same key loaded from a file for testing.

---

## Part 3 — nvim-mirror CLI (next up)

Reads `manifest/plugins.json` and syncs every plugin into Artifactory.

**What it does per plugin:**
1. `git clone --depth 1 <upstream> --branch <branch>` at the pinned SHA
2. Calls `nvim-scan` against the cloned source — aborts if scan fails
3. `tar.gz` the result
4. HTTP PUT to Artifactory generic repo:
   `https://artifactory.company.com/nvim-plugins/<repo>/<sha>.tar.gz`
5. Writes a run report (JSON lines)

**Artifactory path convention:**
```
nvim-plugins/
  folke/lazy.nvim/306a055.tar.gz
  nvim-treesitter/nvim-treesitter/7c14161.tar.gz
  ...
```

**Go concepts introduced:** `os/exec` (git clone), `archive/tar`, `compress/gzip`,
`net/http` (Artifactory REST API PUT), `io` (streaming).

---

## Part 4 — nvim-scan CLI (next up after mirror)

Static analysis of plugin Lua source. Runs inside the mirror pipeline before
a plugin is allowed into Artifactory.

**Checks:**
- Hardcoded URLs (`https?://`) outside comments
- Outbound-capable calls: `io.popen`, `os.execute`, `vim.fn.system`,
  `vim.fn.jobstart` with non-local arguments
- Encoded payloads: `base64`, `load(` with a string argument
- AI config path references: `.claude`, `.copilot`, `.vscode`, `.ssh`
- Strings from `manifest/blocklist.json` (maintained separately)

**Output:** JSON report — pass/fail with findings per file.
Mirror aborts if any finding is severity ERROR.

---

## Part 5-7 — nvim-wrapper (the core security piece)

Replaces the `nvim` binary. Employees type `nvim`, they run this.

**Startup sequence:**
1. Fetch `plugins.json` + `plugins.json.sig` from Artifactory (or use cache)
2. Verify ed25519 signature against public key compiled into binary
3. SHA256-hash all protected files (pre-snapshot)
4. Set protected files immutable
5. Set up OS sandbox (platform-specific)
6. `exec()` real nvim binary — wrapper process becomes nvim

**Exit sequence (deferred, runs after nvim exits):**
1. Remove immutability
2. SHA256-hash protected files (post-snapshot)
3. If pre ≠ post: restore from backup, emit SECURITY_VIOLATION audit event
4. Ship audit log batch to NVIM_AUDIT_ENDPOINT (if set)

**Protected files/dirs:**
```
~/.claude/settings.json
~/.claude/credentials
~/.copilot/
~/.vscode/settings.json
~/.vscode/tasks.json
~/.gitconfig
~/.ssh/config
```

**Platform split:**
- `platform_darwin.go` — `chflags uchg` (immutability), `sandbox-exec` (sandbox)
- `platform_linux.go`  — `chattr +i` via ioctl (immutability), Landlock LSM (sandbox)

**Build-time public key injection:**
```bash
go build -ldflags="-X main.manifestPublicKey=<base64-pubkey>" ./cmd/nvim-wrapper/
```

---

## Manifest Signing Workflow (CI)

```
PR merged to manifest repo
  → CI: nvim-manifest sign /secrets/signing.key
        produces manifest/plugins.json.sig
  → CI: publishes plugins.json + plugins.json.sig to Artifactory
  → CI: builds new managed-nvim release tarball
  → CI: pushes release tarball to Artifactory

Developer machine (next nvim startup):
  → wrapper fetches new plugins.json + .sig from Artifactory
  → verifies signature
  → new whitelist takes effect
```

Private key lives in CI secrets only — never on developer machines, never committed.
