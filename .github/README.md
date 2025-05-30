# GitHub Workflows for GoCreate MCP Server

This directory contains GitHub Actions workflows for the GoCreate MCP (Model Context Protocol) server project.

## Workflows Overview

### 1. CI Workflow (`ci.yml`)
**Triggers:** Push to main/develop, Pull Requests
**Purpose:** Continuous Integration testing across multiple platforms

**Features:**
- **Multi-platform testing:** Ubuntu, macOS, Windows
- **Multi-version Go support:** Go 1.22, 1.23, 1.24
- **Multi-architecture:** AMD64 and ARM64
- **Comprehensive testing:**
  - Unit tests with race detection
  - Benchmarks
  - Code coverage (uploaded to Codecov)
  - Linting with golangci-lint
- **MCP Server validation:**
  - Build verification
  - Basic functionality testing
  - Integration tests with sample Go files

### 2. MCP Server Tests (`mcp-test.yml`)
**Triggers:** Push to main/develop, Pull Requests
**Purpose:** Specialized testing for MCP server functionality

**Features:**
- **MCP Protocol Testing:**
  - Node.js MCP SDK integration tests
  - Stdio transport validation
  - Tool invocation testing (search_code, read_file)
- **Tool Package Testing:**
  - Individual tool package validation
  - Performance benchmarks
  - Race condition detection
  - Coverage reporting

### 3. Security Workflow (`security.yml`)
**Triggers:** Push to main/develop, Pull Requests, Daily schedule (2 AM UTC)
**Purpose:** Security scanning and vulnerability detection

**Features:**
- **Vulnerability Scanning:**
  - `govulncheck` for Go-specific vulnerabilities
  - Nancy for dependency vulnerabilities
  - Trivy for comprehensive security scanning
- **Static Analysis:**
  - Gosec for Go security issues
  - Semgrep for security patterns
  - CodeQL for code analysis
- **Compliance:**
  - Dependency review for PRs
  - License compliance checking
  - SARIF report uploads to GitHub Security tab

### 4. Release Workflow (`release.yml`)
**Triggers:** Git tags starting with 'v' (e.g., v1.0.0)
**Purpose:** Automated release builds and GitHub releases

**Features:**
- **Cross-platform builds:**
  - Linux (AMD64, ARM64)
  - macOS (AMD64, ARM64)
  - Windows (AMD64, ARM64)
- **Release artifacts:**
  - Compressed archives (.tar.gz for Unix, .zip for Windows)
  - SHA256 checksums
  - Automated release notes
- **Version injection:** Binary version from git tag

## Workflow Dependencies

### Required Secrets
- `CODECOV_TOKEN`: For coverage uploads (optional)
- `SEMGREP_APP_TOKEN`: For Semgrep scanning (optional)

### Required Permissions
- `contents: write`: For release creation
- `security-events: write`: For SARIF uploads

## Development Workflow

1. **Feature Development:**
   - Create feature branch
   - CI workflow runs on push
   - MCP tests validate functionality
   - Security scans check for issues

2. **Pull Request:**
   - All workflows run
   - Dependency review checks new dependencies
   - Code coverage and quality gates

3. **Release:**
   - Tag with version (e.g., `git tag v1.0.0`)
   - Push tag to trigger release workflow
   - Automated cross-platform builds
   - GitHub release with artifacts

## Local Testing

Before pushing, run local checks:
```bash
# Run all checks (format, vet, lint, test)
make check

# Run specific tests
make test

# Build for current platform
make build

# Cross-platform build
make build-all
```

## Monitoring

- **CI Status:** Check workflow status on GitHub Actions tab
- **Security:** Monitor GitHub Security tab for vulnerability alerts
- **Coverage:** View coverage reports on Codecov (if configured)
- **Dependencies:** Review dependency updates via Dependabot

## Troubleshooting

### Common Issues

1. **Test Failures:**
   - Check if tests pass locally with `make test`
   - Verify Go version compatibility
   - Check for race conditions with `go test -race`

2. **Security Alerts:**
   - Review SARIF reports in GitHub Security tab
   - Update dependencies with `go mod tidy`
   - Run `govulncheck` locally

3. **Build Failures:**
   - Verify cross-compilation with `make build-all`
   - Check for platform-specific code issues
   - Ensure CGO_ENABLED=0 for static builds

### Workflow Debugging

- Enable debug logging by setting `ACTIONS_STEP_DEBUG=true` in repository secrets
- Check workflow logs for detailed error messages
- Use `act` tool for local workflow testing

## Contributing

When modifying workflows:
1. Test changes in a fork first
2. Validate YAML syntax
3. Check action versions for security updates
4. Update this documentation for significant changes 