package searchdocs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// SupportedVersions represents the structure of the supported versions JSON file
type SupportedVersions struct {
	LastUpdated       string   `json:"lastUpdated"`
	SupportedVersions []string `json:"supportedVersions"`
	LatestVersion     string   `json:"latestVersion"`
}

// LoadSupportedVersions loads the supported enterprise versions from the JSON file
func LoadSupportedVersions() (*SupportedVersions, error) {
	// Get the executable path
	execPath, err := os.Executable()
	if err != nil {
		return nil, err
	}

	// Build path to data directory relative to executable
	dataPath := filepath.Join(filepath.Dir(execPath), "data", "supported-versions.json")

	// If that doesn't exist, try relative to current working directory (for development)
	if _, statErr := os.Stat(dataPath); os.IsNotExist(statErr) {
		dataPath = filepath.Join("data", "supported-versions.json")
	}

	// Read the file
	data, err := os.ReadFile(dataPath)
	if err != nil {
		return nil, err
	}

	// Parse JSON
	var versions SupportedVersions
	if err := json.Unmarshal(data, &versions); err != nil {
		return nil, err
	}

	return &versions, nil
}

// IsVersionSupported checks if a given enterprise server version is supported
func IsVersionSupported(version string) bool {
	versions, err := LoadSupportedVersions()
	if err != nil {
		// Fallback to hardcoded versions if file loading fails
		hardcodedVersions := []string{"3.14", "3.15", "3.16", "3.17"}
		for _, v := range hardcodedVersions {
			if v == version {
				return true
			}
		}
		return false
	}

	for _, v := range versions.SupportedVersions {
		if v == version {
			return true
		}
	}
	return false
}

// NormalizeVersion normalizes version strings for the search API
func NormalizeVersion(v string) string {
	switch v {
	case "free-pro-team", "enterprise-cloud":
		return v
	}

	// Handle enterprise-server versions
	if strings.HasPrefix(v, "enterprise-server@") {
		// Extract version number
		versionPart := strings.TrimPrefix(v, "enterprise-server@")

		// Check if the specific version is supported
		if IsVersionSupported(versionPart) {
			return v
		}

		// If version is not supported, fall back to latest supported version
		versions, err := LoadSupportedVersions()
		if err == nil && versions.LatestVersion != "" {
			return "enterprise-server@" + versions.LatestVersion
		}

		// Ultimate fallback
		return "enterprise-server@3.17"
	}

	return "free-pro-team"
}

// IsLight detects if the terminal is using a light color scheme
func IsLight() bool {
	// Try GH_THEME first (GitHub CLI sets this)
	switch os.Getenv("GH_THEME") {
	case "light":
		return true
	case "dark":
		return false
	}

	// Check COLORFGBG environment variable (set by some terminals)
	// Format is usually "foreground;background" where light background is high numbers
	if colorfgbg := os.Getenv("COLORFGBG"); colorfgbg != "" {
		parts := strings.Split(colorfgbg, ";")
		if len(parts) >= 2 {
			if bg, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
				// Light backgrounds typically have high color numbers (7, 15, etc.)
				return bg >= 7
			}
		}
	}

	// Check for known light terminal programs
	termProgram := os.Getenv("TERM_PROGRAM")
	switch termProgram {
	case "Apple_Terminal":
		// macOS Terminal defaults to light theme
		return true
	case "iTerm.app":
		// iTerm2 - can't reliably detect, assume dark as it's more common
		return false
	case "vscode":
		// VS Code integrated terminal - assume follows editor theme, default dark
		return false
	}

	// Check if we're in a known IDE terminal that might be light
	if os.Getenv("VSCODE_INJECTION") != "" || os.Getenv("TERM_PROGRAM") == "vscode" {
		return false // VS Code defaults to dark
	}

	// Platform-specific defaults
	switch runtime.GOOS {
	case "windows":
		// Windows terminal traditionally light, but newer Windows Terminal is dark
		// Check if we're in newer Windows Terminal
		if os.Getenv("WT_SESSION") != "" {
			return false // Windows Terminal defaults to dark
		}
		return true // Traditional Windows console is light
	case "darwin":
		// macOS Terminal.app defaults to light, but most developers use dark
		return false
	default:
		// Linux and others - most terminals default to dark
		return false
	}
}

// GetTerminalWidth returns the width of the terminal, or a default value if detection fails
func GetTerminalWidth() int {
	// Try to get terminal width from stdout
	if width, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && width > 0 {
		return width
	}

	// Try to get terminal width from stderr as fallback
	if width, _, err := term.GetSize(int(os.Stderr.Fd())); err == nil && width > 0 {
		return width
	}

	// Check COLUMNS environment variable
	if cols := os.Getenv("COLUMNS"); cols != "" {
		if width, err := strconv.Atoi(cols); err == nil && width > 0 {
			return width
		}
	}

	// Default fallback width (matches minimum wrap width used in main.go)
	return 120
}

// Fatal prints an error message and exits with status 1
func Fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
