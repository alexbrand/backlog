# Release Strategy

This document describes the release process for the `backlog` CLI.

## Versioning

This project follows [Semantic Versioning](https://semver.org/):

- **MAJOR** (X.0.0): Breaking changes to CLI commands, flags, or output formats
- **MINOR** (0.X.0): New features, commands, or backward-compatible enhancements
- **PATCH** (0.0.X): Bug fixes and minor improvements

### Pre-release Versions

For testing releases before general availability:

- Alpha: `vX.Y.Z-alpha.N` (early testing, may be unstable)
- Beta: `vX.Y.Z-beta.N` (feature complete, testing for bugs)
- Release Candidate: `vX.Y.Z-rc.N` (final testing before release)

GoReleaser automatically marks these as pre-releases on GitHub.

## Release Process

### Pre-release Checklist

Before creating a release:

1. **Ensure all tests pass**
   ```bash
   make test
   make spec-all
   ```

2. **Verify the build works**
   ```bash
   make build
   ./backlog version
   ```

3. **Review changes since last release**
   ```bash
   git log $(git describe --tags --abbrev=0)..HEAD --oneline
   ```

4. **Update documentation** if any user-facing behavior changed

5. **Ensure main branch is up to date**
   ```bash
   git checkout main
   git pull origin main
   ```

### Creating a Release

1. **Create and push a version tag**
   ```bash
   # For a regular release
   git tag v0.2.0
   git push origin v0.2.0

   # For a pre-release
   git tag v0.2.0-beta.1
   git push origin v0.2.0-beta.1
   ```

2. **Monitor the release workflow**

   The [GoReleaser workflow](.github/workflows/goreleaser.yml) triggers automatically on version tags. Monitor progress at:
   `https://github.com/alexbrand/backlog/actions`

3. **Verify the release**
   - Check the [Releases page](https://github.com/alexbrand/backlog/releases)
   - Verify all binaries are present (darwin/linux × amd64/arm64, windows-amd64)
   - Verify checksums.txt is included
   - Review the auto-generated changelog

## What Gets Released

GoReleaser builds binaries for:

| OS      | Architecture | Binary Name            |
|---------|--------------|------------------------|
| macOS   | ARM64        | backlog-darwin-arm64   |
| macOS   | AMD64        | backlog-darwin-amd64   |
| Linux   | ARM64        | backlog-linux-arm64    |
| Linux   | AMD64        | backlog-linux-amd64    |
| Windows | AMD64        | backlog-windows-amd64  |

Each release includes:
- Platform-specific binaries
- SHA256 checksums (`checksums.txt`)
- Auto-generated changelog (excludes docs, test, ci, chore commits)

## Distribution Channels

### GitHub Releases

Primary distribution. Users can download binaries directly:
```bash
# Example: download and install on Linux
curl -LO https://github.com/alexbrand/backlog/releases/download/v0.2.0/backlog-linux-amd64
chmod +x backlog-linux-amd64
sudo mv backlog-linux-amd64 /usr/local/bin/backlog
```

### Homebrew (macOS/Linux)

After a release, update the Homebrew tap:

1. **Download the new checksums**
   ```bash
   curl -LO https://github.com/alexbrand/backlog/releases/download/vX.Y.Z/checksums.txt
   ```

2. **Update `Formula/backlog.rb`**
   - Update the `version` field
   - Replace SHA256 placeholders with actual checksums from `checksums.txt`

3. **Push to the tap repository**
   ```bash
   # If using a separate tap repo
   cp Formula/backlog.rb /path/to/homebrew-tap/Formula/
   cd /path/to/homebrew-tap
   git add Formula/backlog.rb
   git commit -m "backlog X.Y.Z"
   git push
   ```

Users install via:
```bash
brew install alexbrand/tap/backlog
```

### Go Install

Users can install directly from source:
```bash
go install github.com/alexbrand/backlog/cmd/backlog@latest
go install github.com/alexbrand/backlog/cmd/backlog@v0.2.0
```

## Hotfix Releases

For urgent fixes to a released version:

1. **Create a release branch** (if not already exists)
   ```bash
   git checkout -b release/v0.2.x v0.2.0
   ```

2. **Cherry-pick or apply the fix**
   ```bash
   git cherry-pick <commit-sha>
   # or make the fix directly
   ```

3. **Tag and release**
   ```bash
   git tag v0.2.1
   git push origin release/v0.2.x v0.2.1
   ```

4. **Merge fix back to main** (if applicable)
   ```bash
   git checkout main
   git cherry-pick <commit-sha>
   ```

## Rollback

If a release has critical issues:

1. **Delete the release** from GitHub (keeps the tag for reference)
2. **Create a new patch release** with the fix
3. **Communicate** the issue and resolution to users

Avoid deleting tags as users may have already referenced them.

## Local Testing

Test the release process locally without publishing:

```bash
# Dry run - builds but doesn't publish
goreleaser release --snapshot --clean

# Check the output
ls -la dist/
```

## Changelog

The changelog is auto-generated from commit messages. To ensure good changelogs:

- Use clear, descriptive commit messages
- Prefix with type: `feat:`, `fix:`, `docs:`, `test:`, `ci:`, `chore:`
- Commits prefixed with `docs:`, `test:`, `ci:`, `chore:` are excluded from the changelog

Example commit messages:
```
feat: add support for Linear backend
fix: handle empty task list gracefully
docs: update installation instructions
```
