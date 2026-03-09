# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in nvy, please report it responsibly.

**Do not open a public issue.** Instead, email security concerns to the maintainer or use [GitHub's private vulnerability reporting](https://github.com/trevorphillipscoding/nvy/security/advisories/new).

Please include:

- A description of the vulnerability
- Steps to reproduce the issue
- The potential impact
- Any suggested fixes (if applicable)

You should receive a response within 48 hours. We will work with you to understand the issue and coordinate a fix before any public disclosure.

## Security Model

nvy is designed with security as a priority:

- **HTTPS only** — all downloads are performed over HTTPS. Plain HTTP URLs are rejected, and redirects to HTTP are blocked.
- **TLS 1.2+** — the HTTP client enforces a minimum TLS version of 1.2.
- **SHA-256 verification** — every downloaded archive is verified against its published SHA-256 checksum before extraction. Mismatches abort the install.
- **Zip Slip protection** — archive extraction validates that no entry escapes the destination directory, preventing path-traversal attacks.
- **Decompression bomb protection** — individual extracted files are capped at 2 GB to prevent disk exhaustion.
- **Atomic installs** — installations use a temp directory + rename strategy, ensuring the runtime directory is never in a partial state.
- **Symlink validation** — absolute symlinks and symlinks that escape the destination are rejected during extraction.
- **No shell evaluation** — version files are read as plain text; their contents are never executed.

## Supported Versions

Security fixes are applied to the latest release only. We recommend always running the most recent version of nvy.

| Version | Supported |
| ------- | --------- |
| latest  | Yes       |
| older   | No        |

## Dependencies

nvy has a single external dependency ([cobra](https://github.com/spf13/cobra)) and relies heavily on the Go standard library. We monitor dependencies for known vulnerabilities using `govulncheck` and Dependabot.
