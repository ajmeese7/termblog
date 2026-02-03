# Release Lifecycle

This document describes how to create and publish releases for TermBlog.

## Overview

TermBlog uses [Semantic Versioning](https://semver.org/) and automated releases via GitHub Actions and GoReleaser.

When you push a version tag (e.g., `v0.1.0`), GitHub Actions automatically:
1. Builds binaries for Linux and macOS (amd64 and arm64)
2. Generates a changelog from commit messages
3. Creates a GitHub Release with downloadable assets
4. Publishes checksums for verification

## Version Numbering

Format: `MAJOR.MINOR.PATCH`

| Increment | When to use | Example |
|-----------|-------------|---------|
| **MAJOR** | Breaking changes (config format, CLI args, API) | `0.x.x` → `1.0.0` |
| **MINOR** | New features, non-breaking additions | `0.1.x` → `0.2.0` |
| **PATCH** | Bug fixes, security patches, docs | `0.1.0` → `0.1.1` |

### Pre-1.0 Releases

While in `0.x.x`, minor versions may include breaking changes. After `1.0.0`, the API is considered stable.

## Creating a Release

### Prerequisites

- All changes committed and pushed to `master`
- Tests passing (if applicable)
- Working directory clean

### Quick Release

```bash
# Create and push a release tag
make release VERSION=0.1.0
```

This runs `git tag -a v0.1.0 -m "Release v0.1.0"` and pushes to origin.

### Manual Release

```bash
# 1. Ensure you're on master with latest changes
git checkout master
git pull origin master

# 2. Create an annotated tag
git tag -a v0.1.0 -m "Release v0.1.0"

# 3. Push the tag (triggers GitHub Actions)
git push origin v0.1.0
```

### What Happens Next

1. GitHub Actions detects the new tag
2. GoReleaser builds binaries for all platforms
3. A GitHub Release is created with:
   - Pre-built binaries (tar.gz archives)
   - Auto-generated changelog
   - Installation instructions
   - Checksums file

## Commit Message Convention

We use [Conventional Commits](https://www.conventionalcommits.org/) for automatic changelog generation:

```
<type>(<scope>): <description>

[optional body]
```

### Types

| Type | Description | Changelog Section |
|------|-------------|-------------------|
| `feat` | New feature | Features |
| `fix` | Bug fix | Bug Fixes |
| `docs` | Documentation only | (excluded) |
| `test` | Adding/updating tests | (excluded) |
| `chore` | Maintenance tasks | (excluded) |
| `refactor` | Code restructuring | Other |
| `perf` | Performance improvement | Other |

### Examples

```bash
feat(tui): add vim-style navigation
fix(ssh): handle connection timeout gracefully
docs: update installation instructions
chore: update dependencies
```

## Installation Methods for Users

After a release, users can install TermBlog via:

### Download Binary

```bash
# Linux (amd64)
curl -sSL https://github.com/ajmeese7/termblog/releases/latest/download/termblog_VERSION_linux_amd64.tar.gz | tar xz
sudo mv termblog /usr/local/bin/

# Linux (arm64)
curl -sSL https://github.com/ajmeese7/termblog/releases/latest/download/termblog_VERSION_linux_arm64.tar.gz | tar xz
sudo mv termblog /usr/local/bin/

# macOS (Apple Silicon)
curl -sSL https://github.com/ajmeese7/termblog/releases/latest/download/termblog_VERSION_darwin_arm64.tar.gz | tar xz
sudo mv termblog /usr/local/bin/

# macOS (Intel)
curl -sSL https://github.com/ajmeese7/termblog/releases/latest/download/termblog_VERSION_darwin_amd64.tar.gz | tar xz
sudo mv termblog /usr/local/bin/
```

### Go Install

```bash
# Latest release
go install github.com/ajmeese7/termblog/cmd/termblog@latest

# Specific version
go install github.com/ajmeese7/termblog/cmd/termblog@v0.1.0
```

### From Source

```bash
git clone https://github.com/ajmeese7/termblog.git
cd termblog
make build
```

## Upgrading

### Check Current Version

```bash
termblog version
termblog version --full  # includes commit and build date
```

### Upgrade Methods

**Binary users:** Download the new release and replace the binary.

**Go install users:**
```bash
go install github.com/ajmeese7/termblog/cmd/termblog@latest
```

**Source users:**
```bash
git pull origin master
make build
```

## Troubleshooting

### Release Failed in GitHub Actions

1. Check the [Actions tab](https://github.com/ajmeese7/termblog/actions) for error logs
2. Common issues:
   - GoReleaser config syntax error
   - Missing cross-compilation toolchain
   - Tag not matching expected pattern

### Delete a Bad Tag

```bash
# Delete local tag
git tag -d v0.1.0

# Delete remote tag
git push origin :refs/tags/v0.1.0
```

Then fix the issue and create a new tag.

### Test GoReleaser Locally

```bash
# Install goreleaser
go install github.com/goreleaser/goreleaser/v2@latest

# Test the config (no publish)
goreleaser check
goreleaser release --snapshot --clean
```

## File Reference

| File | Purpose |
|------|---------|
| `Makefile` | Build and release commands |
| `.goreleaser.yaml` | GoReleaser configuration |
| `.github/workflows/release.yml` | GitHub Actions workflow |
| `internal/version/version.go` | Version variables (set at build time) |
