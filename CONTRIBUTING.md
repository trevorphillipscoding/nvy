# Contributing to nvy

Thank you for considering contributing to nvy! This document provides guidelines and instructions.

## Code of Conduct

Be respectful, inclusive, and constructive. We're here to build something useful together.

## How Can I Contribute?

### Reporting Bugs

- Check if the bug has already been reported in [Issues](../../issues)
- Use the bug report template
- Include as much detail as possible: OS, shell, nvy version, steps to reproduce

### Suggesting Features

- Check if the feature has already been requested
- Use the feature request template
- Explain the use case and why it would be valuable

### Adding a New Runtime Plugin

New runtime support is welcome! To add a new runtime (e.g., Ruby, Rust, Deno):

1. Create a new file `plugins/yourruntime/yourruntime.go`
2. Implement the `plugins.Plugin` interface:
   - `Name() string` — canonical runtime name (e.g., `"ruby"`)
   - `Aliases() []string` — alternative names that resolve to this plugin (e.g., `[]string{"rb"}`)
   - `Resolve(version, goos, goarch string) (*plugins.DownloadSpec, error)` — return the download URL, checksum info, and strip depth for the given version and platform
3. Call `plugins.Register(New())` in the package's `init()` function
4. Add a blank import of your package to `plugins/all/all.go`
5. Test thoroughly on macOS and Linux

See existing plugins in `plugins/golang/`, `plugins/node/`, `plugins/python/` for examples.

### Pull Request Process

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/your-feature-name`
3. Make your changes
4. Run tests: `make test`
5. Commit with clear messages
6. Push to your fork and open a pull request
7. The maintainers will review your PR

## Development Setup

### Requirements

- Go 1.25 or later
- macOS or Linux

### Building

```sh
git clone https://github.com/trevorphillipscoding/nvy
cd nvy
make build
./nvy --help
```

### Running Tests

```sh
make test
```

### Code Style

- Follow standard Go conventions
- Run `go fmt` before committing
- Run `make lint` and address issues
- Keep functions focused and well-documented
- Write tests for new functionality

## Project Structure

```
cmd/          — CLI commands (cobra)
internal/     — Core logic (not importable by external code)
  archive/    — Decompression (tar.gz, zip)
  env/        — Environment detection
  fetch/      — HTTP download with progress, checksums
  shim/       — Version resolution and exec dispatch
  state/      — Version state management
plugins/      — Runtime plugins (golang, node, python, etc.)
```

## Questions?

Open an issue or discussion if you have questions about contributing.
