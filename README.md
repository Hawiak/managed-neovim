# managed-nvim

A locked-down Neovim distribution for corporate fleets. Plugins are whitelisted, mirrored through a package registry, and protected against supply-chain attacks.

## Threat model

A whitelisted plugin receives a malicious update that writes hooks into `.claude/settings.json`, `.copilot/`, `.vscode/settings.json`, or uses Neovim autocmds to exfiltrate code or tokens.

Defense: even a fully compromised whitelisted plugin must not be able to do any of this. Enforcement is at the OS level, below Lua.

---

## nvim-wrapper

Replaces the `nvim` binary on developer machines. Employees type `nvim` — this runs instead.

**Startup sequence:**
1. Reads `managed-nvim.toml` from the install dir or `~/.config/managed-nvim/`
2. Fetches `plugins.json` + `plugins.json.sig` from the configured URL, verifies the ed25519 signature, caches to `~/.local/share/managed-nvim/`
3. Hashes protected files (`~/.claude/`, `~/.copilot/`, `~/.vscode/`, `~/.ssh/`, `~/.gitconfig`)
4. Sets those files immutable (`chflags uchg` on macOS, `chattr +i` on Linux)
5. Launches real Neovim inside an OS sandbox
6. On exit: removes immutability, re-hashes, logs any mismatch as a security violation

**Protected files:**
```
~/.claude/settings.json
~/.copilot/
~/.vscode/settings.json
~/.vscode/tasks.json
~/.gitconfig
~/.ssh/config
```

**Building:**
```bash
go build -ldflags="-X main.manifestPublicKey=<base64-pubkey>" ./cmd/nvim-wrapper/
```

The public key is compiled into the binary at build time. Manifests with a mismatched signature are rejected and the cached copy is used instead.

---

## Sandbox

### macOS (`sandbox-exec`)

A `sandbox-exec` profile is generated at runtime and applied to the Neovim process.

**Blocked reads** (private key material):
- `~/.ssh/id_rsa`, `id_ed25519`, `id_ecdsa`, `id_dsa`
- `~/.claude/credentials`
- `~/.aws/credentials`

**Blocked writes** (hook injection prevention, belt-and-suspenders on top of immutability):
- `~/.claude/`, `~/.copilot/`, `~/.vscode/`, `~/.ssh/`, `~/.gitconfig`

**Intentionally allowed:**
- `~/.ssh/config` — git subprocesses spawned by plugins need this
- `SSH_AUTH_SOCK` unix socket — git push over SSH works without exposing the private key
- GPG agent socket — commit signing works

### Linux

`chattr +i` immutability is implemented. Landlock sandbox is a stub — not yet enforced.

### Sandbox diagnostic

```bash
nvim-wrapper --sandbox-check
```

Runs the sandbox profile and reports which operations are correctly blocked and which are correctly allowed. Use this to verify a new deployment.

---

## Lua layer

Four modules load before the user's `init.lua`. All violations are hard errors — Neovim refuses to start rather than silently degrading.

### whitelist.lua

Wraps `lazy.setup()`. Before lazy initialises any plugin, every plugin spec is checked against `plugins.json`. Any plugin not in the manifest produces:

```
[myorg] BLOCKED: 'folke/zen-mode.nvim' is not whitelisted.
Contact your administrator to request approval.
```

### network_shim.lua

Wraps `vim.fn.jobstart` and `vim.fn.system`. Each call is matched against the spawned binary name and checked against the calling plugin's declared permissions in `managed-nvim.toml`.

Permission tokens: `git`, `format`, `build`, `install`, `shell`, `network:localhost`, `network:internet`.

A plugin spawning a binary it has no permission for is blocked and the violation is written to the audit log.

### hooks_blocker.lua

Wraps `vim.api.nvim_create_autocmd`. Any autocmd targeting a write event (`BufWrite`, `BufWritePre`, etc.) on a blocked path pattern (`.claude`, `.copilot`, `.vscode`, `.ssh`, `.aws`) is rejected with a hard error.

### audit.lua

Appends newline-delimited JSON events to `~/.local/share/nvim/managed-nvim-audit.log` (configurable via `MANAGED_NVIM_AUDIT_LOG`). Every blocked action from the network shim and hooks blocker is recorded. Fields: `timestamp`, `plugin`, `action`, `reason`, `target`.

---

## Config reference (`managed-nvim.toml`)

```toml
schema_version = 1

[manifest]
url = "https://artifactory.company.com/managed-neovim/manifest/plugins.json"
signing_key = ""   # base64-encoded ed25519 public key

[audit]
log_path = "~/.local/share/nvim/audit.log"
ship_endpoint = ""          # HTTP endpoint to POST audit events to
ship_on_violation = true    # send immediately on any blocked action

[sandbox]
protected_read  = ["~/.ssh/id_ed25519", "~/.claude/credentials"]
protected_write = ["~/.claude/", "~/.vscode/"]

[permissions]
"neogit" = ["git"]
"copilot.vim" = ["network:internet"]
```

Org distros are generated with this file pre-filled by `nvim-distro-init` in [managed-neovim-admin](../managed-neovim-admin).

---

## Repository layout

```
managed-nvim/
├── cmd/nvim-wrapper/
│   ├── main.go               # startup/exit sequence, config loading
│   ├── manifest_fetch.go     # fetch + verify + cache plugins.json
│   ├── platform_darwin.go    # chflags + sandbox-exec
│   ├── platform_linux.go     # chattr (Landlock stub)
│   └── sandbox_check.go      # --sandbox-check diagnostic
├── internal/
│   ├── manifest/
│   │   └── manifest.go           # Plugin and Manifest types, Load/Save
│   └── luaruntime/
│       ├── runtime.go            # embeds nvim/, extracts to cache on startup
│       └── nvim/                 # Lua config — embedded into binary at build time
│           ├── init.lua          # loads managed layer, then user's config
│           └── lua/managed/
│               ├── whitelist.lua
│               ├── network_shim.lua
│               ├── hooks_blocker.lua
│               └── audit.lua
└── managed-nvim.toml             # local dev config
```
