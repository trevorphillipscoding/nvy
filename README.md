# nvy

[![CI](https://github.com/trevorphillipscoding/nvy/actions/workflows/ci.yml/badge.svg)](https://github.com/trevorphillipscoding/nvy/actions/workflows/ci.yml)
[![License](https://img.shields.io/github/license/trevorphillipscoding/nvy)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/trevorphillipscoding/nvy)](go.mod)

A minimalist, plugin-driven runtime version manager for macOS and Linux.

Install and switch between multiple versions of language runtimes — Go, Node.js, Python, and more — with a single binary built from scratch in Go. One tool to replace pyenv, goenv, nvm, fnm, and the rest.

```
$ nvy install go 1.26.0
downloading go 1.26.0
  45.2 / 67.4 MB (67%)
verifying checksum
  checksum OK
extracting
installed go 1.26.0
  run: nvy global go 1.26.0

$ nvy global go 1.26.0
now using go 1.26.0
  binaries: go, gofmt

$ go version
go version go1.26.0 darwin/arm64

$ nvy local go 1.21.0    # pin this project to an older version
pinned go 1.21.0 in ~/projects/myapp
  written to .go-version
```

---

## Installation

### Homebrew (recommended)

```sh
brew install trevorphillipscoding/tap/nvy
```

or...

### Build from source

Requires Go 1.25+.

```sh
git clone https://github.com/trevorphillipscoding/nvy
cd nvy
make install    # installs to $(go env GOPATH)/bin
```

### Shell setup

nvy places shim symlinks in `~/.nvy/shims/`.

**bash** — add to `~/.bashrc`:

```sh
export PATH="$HOME/.nvy/shims:$PATH"
```

**zsh** — add to `~/.zshrc`:

```sh
export PATH="$HOME/.nvy/shims:$PATH"
```

**fish** — add to `~/.config/fish/config.fish`:

```fish
fish_add_path $HOME/.nvy/shims
```

Then restart your terminal.

---

## Usage

### Install a runtime

```sh
nvy install go 1.26.0
nvy install go@1.26.0        # same thing
nvy install node 20.11.1
nvy install python 3.12
```

The runtime is installed to `~/.nvy/runtimes/<tool>/<version>/`.

### Set a global default

```sh
nvy global go 1.26.0
nvy global node 20.11.1
```

Creates shim symlinks in `~/.nvy/shims/` for every binary provided by that version. When you run `go`, nvy resolves the active version and execs the real binary.

### Pin a version per directory

```sh
cd ~/projects/myapp
nvy local go 1.21.0
```

Writes `1.21.0` to `.go-version` in the current directory. Every time you run `go` from this directory or any subdirectory, nvy uses 1.21.0 instead of the global version.

Version files are plain text — commit them to source control to share with your team:

```sh
git add .go-version
```

Resolution order: **local** (`.go-version` walking up to `/`) → **global** → error.

### List installed versions

```sh
nvy list           # all tools
nvy list go        # one tool
nvy ls             # alias
```

```
go
» 1.22.1  (local)       ← active in this directory (from .go-version)
* 1.21.0  (global)      ← active everywhere else

node
* 20.11.1  (global)
  18.19.0
```

`*` = global default, `»` = local pin for the current directory.

### Uninstall a version

```sh
nvy uninstall go 1.21.0
```

If the version being removed is the active global, its shims are cleaned up automatically.

---

## Supported runtimes

| Tool     | Aliases             | Platforms    | Source                           |
| -------- | ------------------- | ------------ | -------------------------------- |
| `go`     | `golang`            | macOS, Linux | dl.google.com                    |
| `node`   | `nodejs`, `node.js` | macOS, Linux | nodejs.org                       |
| `python` | `python3`, `py`     | macOS, Linux | python-build-standalone (GitHub) |

Supported architectures: `amd64` (x86-64), `arm64` (Apple Silicon / ARM).

Adding a new runtime is straightforward — see [Contributing](#contributing).

---

## How it works

nvy uses a **shim-based execution model** with zero subprocess overhead:

```
~/.nvy/
├── runtimes/<tool>/<version>/   # installed runtime trees
├── shims/                       # symlinks to the nvy binary
├── state/
│   ├── global.json              # active global versions
│   └── owners.json              # binary → tool mapping
└── tmp/                         # staging area for installs
```

1. `nvy global go 1.22.1` creates `~/.nvy/shims/go` → symlink to the `nvy` binary
2. When you run `go build`, your shell resolves `~/.nvy/shims/go` → the `nvy` binary
3. nvy detects it was invoked as `go` (via `os.Args[0]`), resolves the active version, and calls `syscall.Exec` to replace itself with the real Go binary
4. The real binary runs directly — no wrapper process, no signal forwarding, no overhead

Version resolution order: **local** `.go-version` file (walking up from cwd) → **global** version → error with setup instructions.

### Plugin architecture

To add a new runtime (e.g., Ruby, Rust, Deno):

1. Create `plugins/<lang>/<lang>.go` implementing the interface
2. Call `plugins.Register(New())` in the package's `init()` function
3. Add a blank import in `plugins/all/all.go`

That's it. The core install/global/local/list/uninstall commands work automatically.

---

## Security

- **HTTPS only** — plain HTTP is rejected; redirects to HTTP are blocked
- **TLS 1.2+** — enforced on all connections
- **SHA-256 verification** — every archive is verified before extraction
- **Zip Slip protection** — archive entries that escape the destination are rejected
- **Decompression bomb protection** — 2 GB per-file limit during extraction
- **Atomic installs** — temp directory + rename ensures no partial state
- **No shell evaluation** — version files are read as plain text, never executed

See [SECURITY.md](SECURITY.md) for the full security policy and vulnerability reporting.

---

## Environment variables

| Variable  | Default  | Description                     |
| --------- | -------- | ------------------------------- |
| `NVY_DIR` | `~/.nvy` | Override the nvy home directory |

---

## Contributing

Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

```sh
make test          # run tests
make lint          # run linter
make cover-check   # verify coverage threshold
```

---

## License

[MIT](LICENSE)
