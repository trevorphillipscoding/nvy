# nvy

[![License](https://img.shields.io/github/license/trevorphillipscoding/nvy)](LICENSE)

A minimalist, plugin-driven runtime version manager for macOS and Linux.

Install and switch between multiple versions of language runtimes — Go, Node.js, and more — with a single binary built from scratch in Go.

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

---

## Supported runtimes

| Tool     | Aliases             | Platforms    | Official source |
| -------- | ------------------- | ------------ | --------------- |
| `go`     | `golang`            | macOS, Linux | dl.google.com   |
| `node`   | `nodejs`, `node.js` | macOS, Linux | nodejs.org      |
| `python` | `python3`, `py`     | macOS, Linux | python.org      |

Supported architectures: `amd64` (x86-64), `arm64` (Apple Silicon / ARM).

---

## Environment variables

| Variable  | Default  | Description                     |
| --------- | -------- | ------------------------------- |
| `NVY_DIR` | `~/.nvy` | Override the nvy home directory |
