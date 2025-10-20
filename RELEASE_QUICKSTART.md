# Release Quickstart Guide

This guide provides a quick reference for releasing Egg Framework using GoReleaser.

## Prerequisites

1. **Install GoReleaser**
   ```bash
   make tools
   ```

2. **Set GitHub Token**
   ```bash
   export GITHUB_TOKEN=your_github_token_here
   ```
   
   Create a token at: https://github.com/settings/tokens/new (with `repo` scope)

3. **Ensure Git Remote**
   ```bash
   git remote -v
   # Should show: github.com/eggybyte-technology/egg
   ```

## Quick Release (v0.0.1)

### Step 1: Verify Current State

```bash
# Check current status
git status

# Verify tag exists
git tag -l
# Should show: v0.0.1

# View commit
git log --oneline -1
```

### Step 2: Push to GitHub

```bash
# Push main branch
git push origin main

# Push tag
git push origin v0.0.1
```

### Step 3: Test Release Locally (Optional)

```bash
# Test configuration
make release-test

# Build snapshot (local test)
make release-snapshot

# Check output in dist/
ls -la dist/
```

### Step 4: Publish Release

```bash
# Ensure GITHUB_TOKEN is set
echo $GITHUB_TOKEN

# Publish to GitHub
make release-publish
```

This will:
- ✅ Build binaries for Linux, macOS, Windows (amd64, arm64)
- ✅ Create archives (.tar.gz, .zip)
- ✅ Generate SHA256 checksums
- ✅ Create GitHub Release
- ✅ Upload all assets

### Step 5: Verify Release

Visit: https://github.com/eggybyte-technology/egg/releases/tag/v0.0.1

Check:
- Release notes
- Binary downloads for all platforms
- Checksums file

## Available Make Targets

```bash
# Show all available commands
make help

# Development
make build          # Build all modules
make build-cli      # Build CLI tool only
make test           # Run all tests
make lint           # Run linter

# Release Management
make release-test     # Test GoReleaser configuration
make release-snapshot # Build snapshot locally
make tag             # Create new version tag (interactive)
make release-publish # Publish release to GitHub

# Quality
make fmt      # Format code
make vet      # Run go vet
make quality  # Run all quality checks
```

## Platform Binaries

After release, users can download:

### Linux
- `egg_0.0.1_linux_amd64.tar.gz`
- `egg_0.0.1_linux_arm64.tar.gz`

### macOS
- `egg_0.0.1_darwin_amd64.tar.gz`
- `egg_0.0.1_darwin_arm64.tar.gz`

### Windows
- `egg_0.0.1_windows_amd64.zip`

## Go Module Installation

After release, users can install via:

```bash
# Install CLI tool
go install github.com/eggybyte-technology/egg/cli/cmd@v0.0.1

# Use framework modules
go get github.com/eggybyte-technology/egg/core@v0.0.1
go get github.com/eggybyte-technology/egg/runtimex@v0.0.1
go get github.com/eggybyte-technology/egg/connectx@v0.0.1
go get github.com/eggybyte-technology/egg/configx@v0.0.1
go get github.com/eggybyte-technology/egg/obsx@v0.0.1
go get github.com/eggybyte-technology/egg/k8sx@v0.0.1
go get github.com/eggybyte-technology/egg/storex@v0.0.1
```

## Next Release

For future releases (e.g., v0.0.2):

```bash
# 1. Make changes and commit
git add .
git commit -m "feat: add new feature"

# 2. Update CHANGELOG.md
# Edit CHANGELOG.md with new version

# 3. Create tag
git tag -a v0.0.2 -m "Release v0.0.2"

# 4. Push
git push origin main
git push origin v0.0.2

# 5. Publish
export GITHUB_TOKEN=your_token
make release-publish
```

## Troubleshooting

### GoReleaser not found
```bash
make tools
# or
go install github.com/goreleaser/goreleaser/v2@latest
```

### GITHUB_TOKEN not set
```bash
export GITHUB_TOKEN=your_github_token
# Add to ~/.zshrc or ~/.bashrc for persistence
```

### Tag already exists
```bash
# Delete local tag
git tag -d v0.0.1

# Delete remote tag (careful!)
git push origin :refs/tags/v0.0.1

# Recreate
git tag -a v0.0.1 -m "Release v0.0.1"
```

### Release failed
1. Check GitHub token permissions
2. Verify tag is pushed to GitHub
3. Check `.goreleaser.yaml` syntax
4. Run `make release-test` first

## Documentation

- Full Release Guide: [docs/RELEASING.md](docs/RELEASING.md)
- CLI Documentation: [docs/egg-cli.md](docs/egg-cli.md)
- Framework Guide: [docs/guide.md](docs/guide.md)

## Support

- GitHub Issues: https://github.com/eggybyte-technology/egg/issues
- Documentation: https://github.com/eggybyte-technology/egg

---

Built with ❤️ by EggyByte Technology

