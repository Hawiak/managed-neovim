# managed-nvim

A locked-down Neovim distribution for corporate fleets. Plugins are whitelisted, mirrored through a package registry, and protected against supply-chain attacks.

## Threat model

A whitelisted plugin receives a malicious update that writes hooks into `.claude/settings.json`, `.copilot/`, `.vscode/settings.json`, or uses Neovim autocmds to exfiltrate code or tokens.

Defense: even a fully compromised whitelisted plugin must not be able to do any of this. Enforcement is at the OS level, below Lua.

## How it works

Employees type `nvim`. The wrapper runs invisibly (<100ms overhead):

1. Fetches latest signed manifest from the registry (cached, only re-fetches when stale)
2. Verifies manifest signature against the public key compiled into the binary
3. Hashes protected files (`~/.claude/`, `~/.copilot/`, `~/.vscode/`, `~/.ssh/`, `~/.gitconfig`)
4. Sets those files immutable (`chflags uchg` on macOS, `chattr +i` on Linux)
5. Launches Neovim inside an OS sandbox (Landlock on Linux, `sandbox-exec` on macOS)
6. On exit: removes immutability, compares hashes, ships audit log

## Repository layout

```
managed-nvim/
├── cmd/
│   └── nvim-wrapper/         # replaces the nvim binary on developer machines
│       ├── main.go
│       ├── manifest_fetch.go
│       ├── platform_darwin.go
│       └── platform_linux.go
│
├── internal/
│   └── manifest/
│       └── manifest.go       # shared manifest types
│
├── nvim/                     # managed Neovim Lua config
│   ├── init.lua
│   └── lua/managed/
│       ├── whitelist.lua     # hard-errors on non-whitelisted plugins
│       ├── network_shim.lua  # per-plugin network permission enforcement
│       ├── hooks_blocker.lua # clears banned autocmd patterns
│       └── audit.lua         # runtime audit log + HTTP shipper
│
└── managed-nvim.toml         # local dev config (org distros generate their own)
```

## Building

```bash
go build -ldflags="-X main.manifestPublicKey=<base64-pubkey>" ./cmd/nvim-wrapper/
```

## Creating an org distribution

Use [managed-neovim-admin](../managed-neovim-admin) to generate an org-specific distro with a pre-configured install script and manifest.
