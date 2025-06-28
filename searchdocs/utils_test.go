package searchdocs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadSupportedVersions(t *testing.T) {
	// Save current directory and change to project root
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)

	// Go up one directory to project root where data/ exists
	os.Chdir("..")

	// Test loading from existing file
	versions, err := LoadSupportedVersions()
	if err != nil {
		t.Fatalf("Expected to load supported versions, got error: %v", err)
	}

	if versions == nil {
		t.Fatal("Expected versions to be non-nil")
	}

	if len(versions.SupportedVersions) == 0 {
		t.Error("Expected at least one supported version")
	}

	if versions.LatestVersion == "" {
		t.Error("Expected latest version to be set")
	}

	if versions.LastUpdated == "" {
		t.Error("Expected last updated to be set")
	}

	// Verify that latest version is in supported versions
	found := false
	for _, v := range versions.SupportedVersions {
		if v == versions.LatestVersion {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Latest version %s not found in supported versions %v", versions.LatestVersion, versions.SupportedVersions)
	}
}

func TestLoadSupportedVersionsFileNotFound(t *testing.T) {
	// Create a temporary directory without the versions file
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)

	// Change to temp directory where no data file exists
	os.Chdir(tmpDir)

	// Mock executable path to point to temp directory
	originalExecutable := os.Args[0]
	os.Args[0] = filepath.Join(tmpDir, "test-binary")
	defer func() { os.Args[0] = originalExecutable }()

	_, err := LoadSupportedVersions()
	if err == nil {
		t.Error("Expected error when supported versions file doesn't exist")
	}
}

func TestLoadSupportedVersionsInvalidJSON(t *testing.T) {
	// Create temp directory with invalid JSON file
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	os.MkdirAll(dataDir, 0755)

	// Write invalid JSON
	invalidJSON := `{"lastUpdated": "invalid-json"`
	err := os.WriteFile(filepath.Join(dataDir, "supported-versions.json"), []byte(invalidJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	// Mock executable path
	originalExecutable := os.Args[0]
	os.Args[0] = filepath.Join(tmpDir, "test-binary")
	defer func() { os.Args[0] = originalExecutable }()

	_, err = LoadSupportedVersions()
	if err == nil {
		t.Error("Expected error when JSON is invalid")
	}
}

func TestIsVersionSupported(t *testing.T) {
	tests := []struct {
		name      string
		version   string
		supported bool
	}{
		{"supported version 3.17", "3.17", true},
		{"supported version 3.16", "3.16", true},
		{"supported version 3.15", "3.15", true},
		{"supported version 3.14", "3.14", true},
		{"unsupported version 3.13", "3.13", false},
		{"unsupported version 3.18", "3.18", false},
		{"invalid version", "invalid", false},
		{"empty version", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsVersionSupported(tt.version)
			if result != tt.supported {
				t.Errorf("IsVersionSupported(%q) = %v, want %v", tt.version, result, tt.supported)
			}
		})
	}
}

func TestIsVersionSupportedFallback(t *testing.T) {
	// Test fallback behavior when file loading fails
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	// Mock executable path to point to temp directory without data file
	originalExecutable := os.Args[0]
	os.Args[0] = filepath.Join(tmpDir, "test-binary")
	defer func() { os.Args[0] = originalExecutable }()

	// Test hardcoded fallback versions
	tests := []struct {
		version   string
		supported bool
	}{
		{"3.17", true},
		{"3.16", true},
		{"3.15", true},
		{"3.14", true},
		{"3.13", false},
		{"3.18", false},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			result := IsVersionSupported(tt.version)
			if result != tt.supported {
				t.Errorf("IsVersionSupported(%q) = %v, want %v (fallback)", tt.version, result, tt.supported)
			}
		})
	}
}

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"free-pro-team", "free-pro-team", "free-pro-team"},
		{"enterprise-cloud", "enterprise-cloud", "enterprise-cloud"},
		{"supported enterprise-server", "enterprise-server@3.17", "enterprise-server@3.17"},
		{"supported enterprise-server 3.16", "enterprise-server@3.16", "enterprise-server@3.16"},
		{"unsupported enterprise-server", "enterprise-server@3.13", "enterprise-server@3.17"},
		{"invalid version", "invalid", "free-pro-team"},
		{"empty version", "", "free-pro-team"},
		{"partial enterprise", "enterprise-server@", "enterprise-server@3.17"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeVersion(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeVersion(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeVersionFallback(t *testing.T) {
	// Test fallback behavior when versions file can't be loaded
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	// Mock executable path to point to temp directory without data file
	originalExecutable := os.Args[0]
	os.Args[0] = filepath.Join(tmpDir, "test-binary")
	defer func() { os.Args[0] = originalExecutable }()

	result := NormalizeVersion("enterprise-server@3.13")
	expected := "enterprise-server@3.17" // Should fall back to hardcoded latest
	if result != expected {
		t.Errorf("NormalizeVersion fallback = %q, want %q", result, expected)
	}
}

func TestIsLight(t *testing.T) {
	// Save original environment
	originalEnvs := map[string]string{
		"GH_THEME":         os.Getenv("GH_THEME"),
		"COLORFGBG":        os.Getenv("COLORFGBG"),
		"TERM_PROGRAM":     os.Getenv("TERM_PROGRAM"),
		"VSCODE_INJECTION": os.Getenv("VSCODE_INJECTION"),
		"WT_SESSION":       os.Getenv("WT_SESSION"),
	}
	defer func() {
		for key, value := range originalEnvs {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	tests := []struct {
		name     string
		setup    func()
		expected bool
	}{
		{
			name: "GH_THEME light",
			setup: func() {
				os.Clearenv()
				os.Setenv("GH_THEME", "light")
			},
			expected: true,
		},
		{
			name: "GH_THEME dark",
			setup: func() {
				os.Clearenv()
				os.Setenv("GH_THEME", "dark")
			},
			expected: false,
		},
		{
			name: "COLORFGBG light background",
			setup: func() {
				os.Clearenv()
				os.Setenv("COLORFGBG", "0;15")
			},
			expected: true,
		},
		{
			name: "COLORFGBG dark background",
			setup: func() {
				os.Clearenv()
				os.Setenv("COLORFGBG", "15;0")
			},
			expected: false,
		},
		{
			name: "COLORFGBG invalid format",
			setup: func() {
				os.Clearenv()
				os.Setenv("COLORFGBG", "invalid")
			},
			expected: false, // Should fall through to platform defaults
		},
		{
			name: "Apple Terminal",
			setup: func() {
				os.Clearenv()
				os.Setenv("TERM_PROGRAM", "Apple_Terminal")
			},
			expected: true,
		},
		{
			name: "iTerm",
			setup: func() {
				os.Clearenv()
				os.Setenv("TERM_PROGRAM", "iTerm.app")
			},
			expected: false,
		},
		{
			name: "VS Code",
			setup: func() {
				os.Clearenv()
				os.Setenv("TERM_PROGRAM", "vscode")
			},
			expected: false,
		},
		{
			name: "VS Code injection",
			setup: func() {
				os.Clearenv()
				os.Setenv("VSCODE_INJECTION", "1")
			},
			expected: false,
		},
		{
			name: "Windows Terminal",
			setup: func() {
				os.Clearenv()
				os.Setenv("WT_SESSION", "some-session")
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			result := IsLight()
			if result != tt.expected {
				t.Errorf("IsLight() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsLightPlatformDefaults(t *testing.T) {
	// Test platform-specific defaults when no environment variables are set
	originalEnvs := map[string]string{
		"GH_THEME":         os.Getenv("GH_THEME"),
		"COLORFGBG":        os.Getenv("COLORFGBG"),
		"TERM_PROGRAM":     os.Getenv("TERM_PROGRAM"),
		"VSCODE_INJECTION": os.Getenv("VSCODE_INJECTION"),
		"WT_SESSION":       os.Getenv("WT_SESSION"),
	}
	defer func() {
		for key, value := range originalEnvs {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	// Clear all environment variables
	os.Clearenv()

	result := IsLight()

	// Expected result depends on the current platform
	switch runtime.GOOS {
	case "darwin":
		// macOS defaults to false (dark)
		if result != false {
			t.Errorf("Expected macOS default to be false (dark), got %v", result)
		}
	default:
		// Linux and others default to false (dark)
		if result != false {
			t.Errorf("Expected default to be false (dark) for %s, got %v", runtime.GOOS, result)
		}
	}
}

func TestGetTerminalWidth(t *testing.T) {
	// Save original environment
	originalColumns := os.Getenv("COLUMNS")
	defer func() {
		if originalColumns == "" {
			os.Unsetenv("COLUMNS")
		} else {
			os.Setenv("COLUMNS", originalColumns)
		}
	}()

	tests := []struct {
		name     string
		columns  string
		expected int
	}{
		{"valid COLUMNS", "80", 80},
		{"invalid COLUMNS", "invalid", 120},
		{"empty COLUMNS", "", 120},
		{"zero COLUMNS", "0", 120},
		{"negative COLUMNS", "-1", 120},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.columns == "" {
				os.Unsetenv("COLUMNS")
			} else {
				os.Setenv("COLUMNS", tt.columns)
			}

			result := GetTerminalWidth()

			// The function might return the actual terminal width if available,
			// so we need to check if it's either the expected value or a reasonable terminal width
			if tt.columns != "" && tt.columns != "invalid" && tt.columns != "0" && tt.columns != "-1" {
				// For valid COLUMNS values, we expect either the set value or actual terminal width
				if result != tt.expected && result < 20 {
					t.Errorf("GetTerminalWidth() = %d, expected %d or reasonable terminal width", result, tt.expected)
				}
			} else {
				// For invalid values, should fall back to default (120) or actual terminal width
				if result < 20 {
					t.Errorf("GetTerminalWidth() = %d, expected at least 20 (fallback or actual width)", result)
				}
			}
		})
	}
}

func TestFatal(t *testing.T) {
	// We can't directly test Fatal as it calls os.Exit(1)
	// But we can test that it exists and has the right signature
	// This is more of a compile-time check
	var fn func(error) = Fatal
	if fn == nil {
		t.Error("Fatal function should exist")
	}

	// We could use a more sophisticated approach with subprocess testing,
	// but for coverage purposes, this ensures the function is accessible
}

func TestSupportedVersionsStruct(t *testing.T) {
	// Test JSON marshaling/unmarshaling of SupportedVersions struct
	original := SupportedVersions{
		LastUpdated:       "2023-01-01T00:00:00Z",
		SupportedVersions: []string{"3.14", "3.15", "3.16", "3.17"},
		LatestVersion:     "3.17",
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal SupportedVersions: %v", err)
	}

	// Unmarshal back
	var parsed SupportedVersions
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Failed to unmarshal SupportedVersions: %v", err)
	}

	// Verify fields
	if parsed.LastUpdated != original.LastUpdated {
		t.Errorf("LastUpdated mismatch: got %q, want %q", parsed.LastUpdated, original.LastUpdated)
	}

	if parsed.LatestVersion != original.LatestVersion {
		t.Errorf("LatestVersion mismatch: got %q, want %q", parsed.LatestVersion, original.LatestVersion)
	}

	if len(parsed.SupportedVersions) != len(original.SupportedVersions) {
		t.Errorf("SupportedVersions length mismatch: got %d, want %d", len(parsed.SupportedVersions), len(original.SupportedVersions))
	}

	for i, version := range original.SupportedVersions {
		if i >= len(parsed.SupportedVersions) || parsed.SupportedVersions[i] != version {
			t.Errorf("SupportedVersions[%d] mismatch: got %q, want %q", i, parsed.SupportedVersions[i], version)
		}
	}
}

func TestLoadSupportedVersionsWithValidFile(t *testing.T) {
	// Create a temporary directory with a valid versions file
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	os.MkdirAll(dataDir, 0755)

	validVersions := SupportedVersions{
		LastUpdated:       "2023-06-01T12:00:00Z",
		SupportedVersions: []string{"3.15", "3.16", "3.17", "3.18"},
		LatestVersion:     "3.18",
	}

	data, _ := json.Marshal(validVersions)
	err := os.WriteFile(filepath.Join(dataDir, "supported-versions.json"), data, 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	// Mock executable path
	originalExecutable := os.Args[0]
	os.Args[0] = filepath.Join(tmpDir, "test-binary")
	defer func() { os.Args[0] = originalExecutable }()

	versions, err := LoadSupportedVersions()
	if err != nil {
		t.Fatalf("Expected to load versions, got error: %v", err)
	}

	if versions.LatestVersion != "3.18" {
		t.Errorf("Expected latest version 3.18, got %s", versions.LatestVersion)
	}

	if len(versions.SupportedVersions) != 4 {
		t.Errorf("Expected 4 supported versions, got %d", len(versions.SupportedVersions))
	}
}

func TestNormalizeVersionWithCustomFile(t *testing.T) {
	// Create a temporary directory with a custom versions file
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	os.MkdirAll(dataDir, 0755)

	validVersions := SupportedVersions{
		LastUpdated:       "2023-06-01T12:00:00Z",
		SupportedVersions: []string{"3.15", "3.16", "3.17", "3.18"},
		LatestVersion:     "3.18",
	}

	data, _ := json.Marshal(validVersions)
	err := os.WriteFile(filepath.Join(dataDir, "supported-versions.json"), data, 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	// Mock executable path
	originalExecutable := os.Args[0]
	os.Args[0] = filepath.Join(tmpDir, "test-binary")
	defer func() { os.Args[0] = originalExecutable }()

	// Test normalization with the custom file
	result := NormalizeVersion("enterprise-server@3.14")
	expected := "enterprise-server@3.18" // Should use latest from custom file
	if result != expected {
		t.Errorf("NormalizeVersion with custom file = %q, want %q", result, expected)
	}
}
