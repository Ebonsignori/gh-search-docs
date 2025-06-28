# Release Automation Scripts

This directory contains scripts for automating the release process using GitHub Models AI to analyze code changes and determine appropriate version bumps.

## Overview

The release automation system works by:

1. **Analyzing Changes**: Comparing files changed since the last git tag
2. **AI Analysis**: Using GitHub Models (OpenAI o4-mini) to determine:
   - Whether changes warrant a major, minor, or patch version bump
   - Generate release notes based on the changes
3. **Automated Release**: Creating tags and GitHub releases with AI-generated content

## Files

- `analyze-release.js` - Main script that analyzes changes using GitHub Models
- `manual-release.js` - Helper script for manual version overrides
- `package.json` - Node.js dependencies for the scripts

## How It Works

### Automatic Analysis

When triggered via the GitHub Actions workflow, the system:

1. Gets the last git tag to establish a baseline
2. Finds all files changed since that tag
3. Generates diffs for changed files
4. Sends the changes to GitHub Models for analysis
5. Receives back:
   - Recommended version bump (major/minor/patch)
   - Reasoning for the decision
   - Formatted release notes

### Version Bump Logic

The AI follows semantic versioning guidelines:

- **MAJOR**: Breaking changes to CLI interface, removed commands, incompatible changes
- **MINOR**: New features, new commands, backwards-compatible functionality
- **PATCH**: Bug fixes, documentation updates, dependency updates, minor improvements

## Usage

### Automatic (Recommended)

The workflow automatically triggers on pushes to `main` branch (excluding documentation changes):

```yaml
on:
  push:
    branches:
      - main
    paths-ignore:
      - "**.md"
      - ".gitignore" 
      - "LICENSE"
```

### Manual Trigger

You can manually trigger a release via GitHub Actions with optional overrides:

1. Go to Actions → Release workflow
2. Click "Run workflow"
3. Optionally specify:
   - `force_version`: Exact version number (e.g., "1.2.3")
   - `force_bump`: Override bump type (major/minor/patch)

### Tag-Based Release

If you already know the version you want and just need AI-generated release notes:

```bash
git tag v1.2.3
git push origin v1.2.3
```

This will:
- Detect that you've predetermined the version (1.2.3)
- Analyze changes since the last release
- Generate release notes using AI
- Create a GitHub release with the tag and notes
- Skip the version bump analysis (since you've already decided)

**Example**: If you're on `v1.0.0` and push `v1.0.1`, the system recognizes this as a patch release and focuses on generating appropriate release notes.

### Local Testing

To test the analysis locally:

```bash
cd scripts
npm run setup          # One-time setup
# Edit .env file to add your GitHub token
npm test               # Run the standard analysis
npm run test-tag 1.2.3 # Test tag-based release for version 1.2.3
```

Or manually:

```bash
cd scripts
npm install
cp .env.example .env   # Then edit .env with your token
node analyze-release.js
```

This will output the analysis results and create `release-analysis.json`.

### Testing Tag-Based Releases

To test what would happen with a specific tag:

```bash
npm run test-tag 1.2.3  # Test with version 1.2.3
npm run test-tag         # Auto-suggest next patch version
```

## Environment Variables

The scripts support both environment variables and `.env` file configuration:

### Required Variables
- `GITHUB_TOKEN`: Required for GitHub Models API access

### Optional Variables  
- `GITHUB_MODELS_ENDPOINT`: Override API endpoint (default: `https://models.github.ai/inference`)
- `GITHUB_MODELS_MODEL`: Override model name (default: `openai/o4-mini`)
- `GITHUB_OUTPUT`: Set by GitHub Actions for output handling

### Local Development Setup

1. **Quick Setup**: `npm run setup` (creates .env from template)
2. **Manual Setup**: 
   ```bash
   cp .env.example .env
   # Edit .env and add your GitHub token
   ```

Example `.env` file:
```
GITHUB_TOKEN=your_github_token_here
# GITHUB_MODELS_ENDPOINT=https://models.github.ai/inference
# GITHUB_MODELS_MODEL=openai/o4-mini
```

## GitHub Models Configuration

The script uses:
- **Endpoint**: `https://models.github.ai/inference`
- **Model**: `openai/o4-mini`
- **Authentication**: GitHub Personal Access Token

## Output

The script generates:

1. **Console Output**: Human-readable analysis results
2. **release-analysis.json**: Structured data for workflow consumption
3. **GitHub Actions Outputs**: For workflow decision making

Example output structure:

```json
{
  "current_version": "1.2.3",
  "new_version": "1.3.0", 
  "version_bump": "minor",
  "reasoning": "Added new CLI command for documentation search",
  "release_notes": "## What's Changed\n\n- Added `ask` command for interactive documentation queries\n- Improved error handling for network requests",
  "changed_files": ["main.go", "cmd/ask.go"]
}
```

## Error Handling

The system handles various error conditions:

- **No previous tags**: Treats as initial release, analyzes all files
- **No changes**: Skips release creation
- **API failures**: Falls back to manual release process
- **Invalid responses**: Logs errors and exits gracefully

## Dependencies

- `openai`: ^4.0.0 - For GitHub Models API access
- `dotenv`: ^16.0.0 - For environment variable management
- Node.js 18+ - Runtime requirement

## Quick Start

1. **Setup**: `cd scripts && npm run setup`
2. **Configure**: Edit `.env` file with your GitHub token
3. **Test**: `npm test` (automatic analysis) or `npm run test-tag 1.2.3` (tag-based)
4. **Use**: 
   - Push to main (automatic analysis)
   - Push a tag (predetermined version)
   - Trigger workflow manually

## Workflow Types

### 1. Automatic Analysis (Recommended for regular development)
- Push changes to `main`
- AI analyzes changes and determines version bump
- Creates tag and release automatically

### 2. Tag-Based Release (When you know the version)
- Useful when you want to control the exact version
- Great for hotfixes, coordinated releases, or when following a release schedule
- AI focuses on generating good release notes

### 3. Manual Override (For special cases)
- Force specific version numbers
- Override version bump decisions
- Emergency releases

## Security Notes

- Uses GitHub token for API access (no additional API keys needed)
- Token permissions are scoped to repository access
- No sensitive data is logged or stored