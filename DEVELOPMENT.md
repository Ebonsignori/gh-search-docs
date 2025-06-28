## Development

You can run this as a regular Go CLI executable without the `gh` CLI:

```bash
git clone https://github.com/<your-username>/gh-search-docs
cd gh-search-docs
go build -o gh-search-docs
./gh-search-docs --size 5 "pull request"
```

### Development Setup

Install development dependencies:

```bash
make dev-setup
```

### Debug Mode

Use `--debug` to see the raw API response for troubleshooting:

```bash
gh search-docs --debug "your query"
```


### Manually Testing the Extension

```bash
# Test basic search
./gh-search-docs "GitHub Actions"

# Test JSON output
./gh-search-docs --format json --size 3 "webhooks"

# Test Enterprise Server
./gh-search-docs --version enterprise-server@3.17 "SAML"

# Test with highlights and includes
./gh-search-docs --highlights title,content --include intro,headings "API authentication"
```

### Common Development Tasks

```bash
# Run tests
make test

# Run tests with coverage
make test-coverage

# Run linter
make lint

# Format code
make fmt

# Run all checks (format, vet, lint, test)
make check

# Build binary
make build

# Clean build artifacts
make clean
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with race detection and coverage
go test -race -coverprofile=coverage.out -covermode=atomic ./...
```

### Releasing

Releasing is controlled manually and uses an LLM to analyze changes and generate release notes. There are three ways to trigger a release:

#### Manual Release via GitHub Actions

1. Go to the [Actions tab](https://github.com/Ebonsignori/gh-search-docs/actions/workflows/release.yml) in GitHub
2. Click "Run workflow"
3. Choose one of these options:
   - **Force a specific version**: Enter a version like `1.2.3` to release that exact version
   - **Force a version bump type**: Select `major`, `minor`, or `patch` to bump from the current version
   - **Let AI analyze**: Leave both fields empty to let the AI analyze recent changes and determine the appropriate version bump

#### Tag-based Release

Push a tag starting with `v` to trigger a release for that specific version:

```bash
git tag v1.2.3
git push origin v1.2.3
```

#### AI Analysis

When using AI analysis (manual trigger with no inputs), the system will:
- Analyze commits since the last release
- Determine if changes warrant a `major`, `minor`, or `patch` version bump
- Generate detailed release notes
- Only create a release if significant changes are detected

The release process automatically runs tests and linting before creating the release.
