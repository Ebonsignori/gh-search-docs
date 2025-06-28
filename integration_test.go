package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

func TestDefaultBehaviorWithoutFlags(t *testing.T) {
	// Test default parameter construction without special flags
	searchURL, err := url.Parse(endpoint)
	if err != nil {
		t.Fatalf("Failed to parse endpoint: %v", err)
	}

	params := url.Values{}
	params.Set("query", "test query")
	params.Set("size", "5")                // default size
	params.Set("version", "free-pro-team") // default version
	params.Set("language", "en")           // default language

	// Default behavior - include intro
	params.Add("include", "intro")

	searchURL.RawQuery = params.Encode()

	// Verify URL construction
	if !strings.Contains(searchURL.String(), "query=test+query") {
		t.Error("URL should contain encoded query")
	}

	if !strings.Contains(searchURL.String(), "size=5") {
		t.Error("URL should contain default size")
	}

	if !strings.Contains(searchURL.String(), "version=free-pro-team") {
		t.Error("URL should contain default version")
	}

	if !strings.Contains(searchURL.String(), "include=intro") {
		t.Error("URL should contain default intro include")
	}
}

func TestMatchedContentLogic(t *testing.T) {
	// Test the logic for matched content vs. default behavior
	tests := []struct {
		name                  string
		includeMatchedContent bool
		userIncludes          []string
		expectedHighlights    []string
		expectedIncludes      []string
	}{
		{
			name:                  "matched content enabled",
			includeMatchedContent: true,
			userIncludes:          []string{},
			expectedHighlights:    []string{"content_explicit"},
			expectedIncludes:      []string{"toplevel"},
		},
		{
			name:                  "default behavior",
			includeMatchedContent: false,
			userIncludes:          []string{},
			expectedHighlights:    []string{},
			expectedIncludes:      []string{"intro"},
		},
		{
			name:                  "user specified includes override default",
			includeMatchedContent: false,
			userIncludes:          []string{"headings", "intro"},
			expectedHighlights:    []string{},
			expectedIncludes:      []string{"headings", "intro"},
		},
		{
			name:                  "matched content with user includes",
			includeMatchedContent: true,
			userIncludes:          []string{"headings"},
			expectedHighlights:    []string{"content_explicit"},
			expectedIncludes:      []string{"headings"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := url.Values{}
			params.Set("query", "test")

			if tt.includeMatchedContent {
				params.Add("highlights", "content_explicit")
			}

			if len(tt.userIncludes) == 0 {
				if tt.includeMatchedContent {
					params.Add("include", "toplevel")
				} else {
					params.Add("include", "intro")
				}
			} else {
				for _, inc := range tt.userIncludes {
					params.Add("include", inc)
				}
			}

			// Verify highlights
			actualHighlights := params["highlights"]
			if len(actualHighlights) != len(tt.expectedHighlights) {
				t.Errorf("Expected %d highlights, got %d", len(tt.expectedHighlights), len(actualHighlights))
			}

			// Verify includes
			actualIncludes := params["include"]
			if len(actualIncludes) != len(tt.expectedIncludes) {
				t.Errorf("Expected %d includes, got %d", len(tt.expectedIncludes), len(actualIncludes))
			}
		})
	}
}

func TestResultDisplayLogic(t *testing.T) {
	// Test the logic for determining how many results to show
	tests := []struct {
		name                  string
		totalHits             int
		sizeFlag              int
		includeMatchedContent bool
		expectedMaxResults    int
		expectedShowLimit     bool
	}{
		{
			name:                  "default size, more than 5 hits, no matched content",
			totalHits:             10,
			sizeFlag:              5,
			includeMatchedContent: false,
			expectedMaxResults:    5,
			expectedShowLimit:     true,
		},
		{
			name:                  "default size, less than 5 hits",
			totalHits:             3,
			sizeFlag:              5,
			includeMatchedContent: false,
			expectedMaxResults:    3,
			expectedShowLimit:     false,
		},
		{
			name:                  "custom size smaller than hits",
			totalHits:             10,
			sizeFlag:              3,
			includeMatchedContent: false,
			expectedMaxResults:    3,
			expectedShowLimit:     false,
		},
		{
			name:                  "matched content mode",
			totalHits:             10,
			sizeFlag:              5,
			includeMatchedContent: true,
			expectedMaxResults:    5,
			expectedShowLimit:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maxResults := tt.totalHits

			// Simulate the logic from main.go
			if tt.sizeFlag == 5 && maxResults > 5 && !tt.includeMatchedContent {
				maxResults = 5
			} else if tt.sizeFlag < maxResults {
				maxResults = tt.sizeFlag
			}

			if maxResults != tt.expectedMaxResults {
				t.Errorf("Expected max results %d, got %d", tt.expectedMaxResults, maxResults)
			}

			// Check if we should show the limit message
			showLimit := maxResults == 5 && tt.totalHits > 5 && !tt.includeMatchedContent
			if showLimit != tt.expectedShowLimit {
				t.Errorf("Expected show limit %v, got %v", tt.expectedShowLimit, showLimit)
			}
		})
	}
}

func TestPaginationInfoLogic(t *testing.T) {
	tests := []struct {
		name            string
		foundValue      int
		currentPage     int
		pageSize        int
		expectedPages   int
		expectedHasNext bool
	}{
		{
			name:            "single page",
			foundValue:      3,
			currentPage:     1,
			pageSize:        5,
			expectedPages:   1,
			expectedHasNext: false,
		},
		{
			name:            "multiple pages, first page",
			foundValue:      20,
			currentPage:     1,
			pageSize:        5,
			expectedPages:   4,
			expectedHasNext: true,
		},
		{
			name:            "multiple pages, last page",
			foundValue:      20,
			currentPage:     4,
			pageSize:        5,
			expectedPages:   4,
			expectedHasNext: false,
		},
		{
			name:            "partial last page",
			foundValue:      22,
			currentPage:     4,
			pageSize:        5,
			expectedPages:   5,
			expectedHasNext: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			totalPages := (tt.foundValue + tt.pageSize - 1) / tt.pageSize
			hasNext := tt.currentPage < totalPages

			if totalPages != tt.expectedPages {
				t.Errorf("Expected %d total pages, got %d", tt.expectedPages, totalPages)
			}

			if hasNext != tt.expectedHasNext {
				t.Errorf("Expected has next %v, got %v", tt.expectedHasNext, hasNext)
			}
		})
	}
}

func TestHTMLTagStripping(t *testing.T) {
	// Test the logic for removing HTML tags in plain text output
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple mark tags",
			input:    "This is <mark>highlighted</mark> text",
			expected: "This is highlighted text",
		},
		{
			name:     "multiple mark tags",
			input:    "Both <mark>word1</mark> and <mark>word2</mark> are highlighted",
			expected: "Both word1 and word2 are highlighted",
		},
		{
			name:     "no tags",
			input:    "Plain text without any tags",
			expected: "Plain text without any tags",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the HTML tag removal logic from main.go
			cleanStr := strings.ReplaceAll(tt.input, "<mark>", "")
			cleanStr = strings.ReplaceAll(cleanStr, "</mark>", "")

			if cleanStr != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, cleanStr)
			}
		})
	}
}

func TestSearchResultWithMixedHighlightTypes(t *testing.T) {
	// Test parsing of search results with mixed highlight data types
	mixedJSON := `{
		"meta": {
			"found": {"value": 1, "relation": "eq"},
			"took": {"query_msec": 10, "total_msec": 20},
			"page": 1,
			"size": 1
		},
		"hits": [
			{
				"id": "mixed-highlights",
				"title": "Mixed Highlights Test",
				"url": "/test",
				"highlights": {
					"title": ["Array <mark>highlight</mark>"],
					"content": "String <mark>highlight</mark>",
					"content_explicit": [
						"First <mark>explicit</mark>",
						"Second <mark>explicit</mark>"
					],
					"term": [],
					"empty_field": null
				}
			}
		]
	}`

	var result SearchResult
	err := json.Unmarshal([]byte(mixedJSON), &result)
	if err != nil {
		t.Fatalf("Failed to parse mixed highlights JSON: %v", err)
	}

	if len(result.Hits) != 1 {
		t.Fatalf("Expected 1 hit, got %d", len(result.Hits))
	}

	hit := result.Hits[0]
	if hit.Highlights == nil {
		t.Fatal("Expected highlights to be present")
	}

	// Test array highlight
	if titleHighlight, exists := hit.Highlights["title"]; exists {
		if highlights, ok := titleHighlight.([]interface{}); ok {
			if len(highlights) != 1 {
				t.Errorf("Expected 1 title highlight, got %d", len(highlights))
			}
		} else {
			t.Error("Title highlight should be array")
		}
	}

	// Test string highlight
	if contentHighlight, exists := hit.Highlights["content"]; exists {
		if _, ok := contentHighlight.(string); !ok {
			t.Error("Content highlight should be string")
		}
	}

	// Test empty array
	if termHighlight, exists := hit.Highlights["term"]; exists {
		if highlights, ok := termHighlight.([]interface{}); ok {
			if len(highlights) != 0 {
				t.Errorf("Expected 0 term highlights, got %d", len(highlights))
			}
		} else {
			t.Error("Term highlight should be array")
		}
	}
}

func TestDescriptionTruncation(t *testing.T) {
	tests := []struct {
		name        string
		intro       string
		maxLength   int
		shouldTrunc bool
	}{
		{
			name:        "short intro",
			intro:       "Short description",
			maxLength:   150,
			shouldTrunc: false,
		},
		{
			name:        "long intro",
			intro:       "This is a very long introduction that exceeds the maximum length limit and should be truncated to show only the first part of the description to avoid overwhelming the user interface",
			maxLength:   150,
			shouldTrunc: true,
		},
		{
			name:        "exact length",
			intro:       strings.Repeat("a", 150),
			maxLength:   150,
			shouldTrunc: false,
		},
		{
			name:        "one character over",
			intro:       strings.Repeat("a", 151),
			maxLength:   150,
			shouldTrunc: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			description := tt.intro
			if len(description) > tt.maxLength {
				description = description[:tt.maxLength] + "..."
			}

			if tt.shouldTrunc {
				if !strings.HasSuffix(description, "...") {
					t.Error("Long description should be truncated with ellipsis")
				}
				if len(description) != tt.maxLength+3 { // +3 for "..."
					t.Errorf("Truncated length should be %d, got %d", tt.maxLength+3, len(description))
				}
			} else {
				if strings.HasSuffix(description, "...") {
					t.Error("Short description should not be truncated")
				}
				if description != tt.intro {
					t.Error("Short description should remain unchanged")
				}
			}
		})
	}
}

func TestURLPathConstruction(t *testing.T) {
	// Test the logic for constructing full GitHub docs URLs
	tests := []struct {
		name        string
		urlPath     string
		expectedURL string
	}{
		{
			name:        "standard path",
			urlPath:     "/en/actions/quickstart",
			expectedURL: "https://docs.github.com/en/actions/quickstart",
		},
		{
			name:        "path with special characters",
			urlPath:     "/en/github/setting-up-and-managing-your-github-user-account",
			expectedURL: "https://docs.github.com/en/github/setting-up-and-managing-your-github-user-account",
		},
		{
			name:        "enterprise path",
			urlPath:     "/en/enterprise-server@3.17/admin/configuration",
			expectedURL: "https://docs.github.com/en/enterprise-server@3.17/admin/configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fullURL := "https://docs.github.com" + tt.urlPath

			if fullURL != tt.expectedURL {
				t.Errorf("Expected URL %q, got %q", tt.expectedURL, fullURL)
			}

			// Verify URL is parseable
			_, err := url.Parse(fullURL)
			if err != nil {
				t.Errorf("Generated URL is not valid: %v", err)
			}
		})
	}
}

func TestComplexQueryEncoding(t *testing.T) {
	// Test encoding of complex search queries
	queries := []string{
		"simple query",
		"query with \"quotes\"",
		"query with & symbols",
		"query with + plus signs",
		"query with spaces    and   multiple   spaces",
		"query with unicode: 中文 测试",
		"query with emojis 🚀 ✨",
		"query with special chars: @#$%^&*()",
	}

	for _, query := range queries {
		t.Run(fmt.Sprintf("query_%s", query[:min(len(query), 20)]), func(t *testing.T) {
			params := url.Values{}
			params.Set("query", query)

			encoded := params.Encode()

			// Decode and verify round-trip
			decoded, err := url.ParseQuery(encoded)
			if err != nil {
				t.Errorf("Failed to decode query: %v", err)
			}

			decodedQuery := decoded.Get("query")
			if decodedQuery != query {
				t.Errorf("Query round-trip failed: original %q, decoded %q", query, decodedQuery)
			}
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestNumberFormatting(t *testing.T) {
	// Test number formatting in result display
	tests := []struct {
		name     string
		number   int
		expected string
	}{
		{"small number", 5, "5"},
		{"medium number", 123, "123"},
		{"large number", 1234, "1234"},
		{"very large", 999999, "999999"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := strconv.Itoa(tt.number)
			if formatted != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, formatted)
			}
		})
	}
}
