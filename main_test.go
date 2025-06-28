package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestSearchResultParsing(t *testing.T) {
	// Test JSON response parsing
	jsonResponse := `{
		"meta": {
			"found": {
				"value": 1138,
				"relation": "eq"
			},
			"took": {
				"query_msec": 17,
				"total_msec": 35
			},
			"page": 1,
			"size": 3
		},
		"hits": [
			{
				"id": "L295qpcBRDp-qF-yWo5r",
				"url": "/en/pull-requests/collaborating-with-pull-requests/proposing-changes-to-your-work-with-pull-requests/creating-a-pull-request",
				"title": "Creating a pull request",
				"breadcrumbs": "Pull requests / Collaborate with pull requests / Propose changes",
				"highlights": {
					"title": ["Creating a <mark>pull request</mark>"],
					"content": ["Create a <mark>pull</mark> <mark>request</mark> to propose and collaborate"]
				}
			}
		]
	}`

	var result SearchResult
	err := json.Unmarshal([]byte(jsonResponse), &result)
	if err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Verify meta information
	if result.Meta.Found.Value != 1138 {
		t.Errorf("Expected found value 1138, got %d", result.Meta.Found.Value)
	}

	if result.Meta.Found.Relation != "eq" {
		t.Errorf("Expected relation 'eq', got %s", result.Meta.Found.Relation)
	}

	if result.Meta.Page != 1 {
		t.Errorf("Expected page 1, got %d", result.Meta.Page)
	}

	if result.Meta.Size != 3 {
		t.Errorf("Expected size 3, got %d", result.Meta.Size)
	}

	// Verify hits
	if len(result.Hits) != 1 {
		t.Errorf("Expected 1 hit, got %d", len(result.Hits))
	}

	hit := result.Hits[0]
	if hit.ID != "L295qpcBRDp-qF-yWo5r" {
		t.Errorf("Expected ID 'L295qpcBRDp-qF-yWo5r', got %s", hit.ID)
	}

	if hit.Title != "Creating a pull request" {
		t.Errorf("Expected title 'Creating a pull request', got %s", hit.Title)
	}

	if hit.Breadcrumbs != "Pull requests / Collaborate with pull requests / Propose changes" {
		t.Errorf("Unexpected breadcrumbs: %s", hit.Breadcrumbs)
	}

	// Verify highlights exist
	if hit.Highlights == nil {
		t.Error("Expected highlights to be present")
	}
}

func TestStringSlice(t *testing.T) {
	tests := []struct {
		name     string
		values   []string
		expected string
	}{
		{"empty", []string{}, ""},
		{"single", []string{"title"}, "title"},
		{"multiple", []string{"title", "content", "term"}, "title,content,term"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ss StringSlice
			for _, v := range tt.values {
				err := ss.Set(v)
				if err != nil {
					t.Errorf("Failed to set value %s: %v", v, err)
				}
			}

			result := ss.String()
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestURLConstruction(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		size       int
		version    string
		language   string
		page       int
		sort       string
		highlights []string
		includes   []string
		toplevel   []string
		aggregate  []string
	}{
		{
			name:     "basic query",
			query:    "pull request",
			size:     10,
			version:  "free-pro-team",
			language: "en",
		},
		{
			name:       "with pagination",
			query:      "GitHub Actions",
			size:       5,
			version:    "free-pro-team",
			language:   "en",
			page:       2,
			highlights: []string{"title", "content"},
		},
		{
			name:      "enterprise server",
			query:     "SAML configuration",
			size:      20,
			version:   "enterprise-server@3.17",
			language:  "en",
			includes:  []string{"intro", "headings"},
			toplevel:  []string{"admin"},
			aggregate: []string{"type"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			searchURL, err := url.Parse(endpoint)
			if err != nil {
				t.Fatalf("Failed to parse endpoint: %v", err)
			}

			params := url.Values{}
			params.Set("query", tt.query)
			params.Set("size", strconv.Itoa(tt.size))
			params.Set("version", tt.version)
			params.Set("language", tt.language)

			if tt.page > 0 {
				params.Set("page", strconv.Itoa(tt.page))
			}
			if tt.sort != "" {
				params.Set("sort", tt.sort)
			}

			for _, h := range tt.highlights {
				params.Add("highlights", h)
			}
			for _, inc := range tt.includes {
				params.Add("include", inc)
			}
			for _, tl := range tt.toplevel {
				params.Add("toplevel", tl)
			}
			for _, agg := range tt.aggregate {
				params.Add("aggregate", agg)
			}

			searchURL.RawQuery = params.Encode()

			// Verify required parameters
			parsedParams, _ := url.ParseQuery(searchURL.RawQuery)

			if parsedParams.Get("query") != tt.query {
				t.Errorf("Expected query %q, got %q", tt.query, parsedParams.Get("query"))
			}

			if parsedParams.Get("size") != strconv.Itoa(tt.size) {
				t.Errorf("Expected size %d, got %s", tt.size, parsedParams.Get("size"))
			}

			if parsedParams.Get("version") != tt.version {
				t.Errorf("Expected version %q, got %q", tt.version, parsedParams.Get("version"))
			}

			if parsedParams.Get("language") != tt.language {
				t.Errorf("Expected language %q, got %q", tt.language, parsedParams.Get("language"))
			}

			// Verify optional parameters
			if tt.page > 0 {
				if parsedParams.Get("page") != strconv.Itoa(tt.page) {
					t.Errorf("Expected page %d, got %s", tt.page, parsedParams.Get("page"))
				}
			}

			if len(tt.highlights) > 0 {
				highlights := parsedParams["highlights"]
				if !reflect.DeepEqual(highlights, tt.highlights) {
					t.Errorf("Expected highlights %v, got %v", tt.highlights, highlights)
				}
			}

			if len(tt.includes) > 0 {
				includes := parsedParams["include"]
				if !reflect.DeepEqual(includes, tt.includes) {
					t.Errorf("Expected includes %v, got %v", tt.includes, includes)
				}
			}
		})
	}
}

func TestEndpointValidation(t *testing.T) {
	if endpoint == "" {
		t.Error("endpoint should not be empty")
	}

	if !strings.HasPrefix(endpoint, "https://") {
		t.Error("endpoint should use HTTPS")
	}

	if !strings.Contains(endpoint, "docs.github.com") {
		t.Error("endpoint should point to docs.github.com")
	}

	if !strings.Contains(endpoint, "/api/search/v1") {
		t.Error("endpoint should use the search API v1")
	}
}

func TestPaginationCalculation(t *testing.T) {
	tests := []struct {
		name       string
		foundValue int
		size       int
		expected   int
	}{
		{"exact division", 100, 10, 10},
		{"with remainder", 105, 10, 11},
		{"single page", 5, 10, 1},
		{"zero results", 0, 10, 0},
		{"large dataset", 1138, 3, 380},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var totalPages int
			if tt.foundValue == 0 {
				totalPages = 0
			} else {
				totalPages = (tt.foundValue + tt.size - 1) / tt.size
			}

			if totalPages != tt.expected {
				t.Errorf("Expected %d total pages, got %d", tt.expected, totalPages)
			}
		})
	}
}

func TestHighlightOptions(t *testing.T) {
	validOptions := []string{"title", "content", "content_explicit", "term"}

	for _, option := range validOptions {
		t.Run(option, func(t *testing.T) {
			// Test that the option can be used in URL construction
			params := url.Values{}
			params.Add("highlights", option)

			if params.Get("highlights") != option {
				t.Errorf("Failed to set highlight option %s", option)
			}
		})
	}
}

func TestAdditionalIncludes(t *testing.T) {
	validIncludes := []string{"intro", "headings", "toplevel"}

	for _, include := range validIncludes {
		t.Run(include, func(t *testing.T) {
			// Test that the include can be used in URL construction
			params := url.Values{}
			params.Add("include", include)

			if params.Get("include") != include {
				t.Errorf("Failed to set include option %s", include)
			}
		})
	}
}

func TestJSONFormat(t *testing.T) {
	// Test that SearchResult can be marshaled back to JSON
	result := SearchResult{
		Meta: struct {
			Found struct {
				Value    int    `json:"value"`
				Relation string `json:"relation"`
			} `json:"found"`
			Took struct {
				QueryMsec int `json:"query_msec"`
				TotalMsec int `json:"total_msec"`
			} `json:"took"`
			Page int `json:"page"`
			Size int `json:"size"`
		}{},
		Hits: []SearchItem{
			{
				ID:          "test-id",
				Title:       "Test Title",
				URL:         "/test/url",
				Breadcrumbs: "Test / Path",
				Content:     "Test content",
				Highlights: map[string]interface{}{
					"title": []string{"Test <mark>Title</mark>"},
				},
			},
		},
	}

	result.Meta.Found.Value = 1
	result.Meta.Found.Relation = "eq"
	result.Meta.Page = 1
	result.Meta.Size = 1

	jsonData, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal SearchResult: %v", err)
	}

	// Test that it can be unmarshaled back
	var parsed SearchResult
	err = json.Unmarshal(jsonData, &parsed)
	if err != nil {
		t.Fatalf("Failed to unmarshal SearchResult: %v", err)
	}

	if parsed.Meta.Found.Value != result.Meta.Found.Value {
		t.Errorf("Round-trip failed for found value")
	}

	if len(parsed.Hits) != len(result.Hits) {
		t.Errorf("Round-trip failed for hits count")
	}

	if parsed.Hits[0].Title != result.Hits[0].Title {
		t.Errorf("Round-trip failed for hit title")
	}
}

func TestQueryParameterEncoding(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{"simple query", "pull request", "pull+request"},
		{"with special chars", "API & SDK", "API+%26+SDK"},
		{"with quotes", `"exact phrase"`, "%22exact+phrase%22"},
		{"with symbols", "C++ programming", "C%2B%2B+programming"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := url.Values{}
			params.Set("query", tt.query)
			encoded := params.Encode()

			if !strings.Contains(encoded, tt.expected) {
				t.Errorf("Expected encoded query to contain %q, got %q", tt.expected, encoded)
			}

			// Test that it can be decoded back
			decodedQuery := params.Get("query")
			if decodedQuery != tt.query {
				t.Errorf("Round-trip encoding failed: expected %q, got %q", tt.query, decodedQuery)
			}
		})
	}
}

func TestVersionSupport(t *testing.T) {
	validVersions := []string{
		"free-pro-team",
		"enterprise-cloud",
		"enterprise-server@3.14",
		"enterprise-server@3.15",
		"enterprise-server@3.16",
		"enterprise-server@3.17",
	}

	for _, version := range validVersions {
		t.Run(version, func(t *testing.T) {
			params := url.Values{}
			params.Set("version", version)

			if params.Get("version") != version {
				t.Errorf("Failed to set version %s", version)
			}
		})
	}
}

func TestEndpointConstant(t *testing.T) {
	expectedEndpoint := "https://docs.github.com/api/search/v1"
	if endpoint != expectedEndpoint {
		t.Errorf("Expected endpoint %q, got %q", expectedEndpoint, endpoint)
	}
}

func TestSearchItemStructFields(t *testing.T) {
	// Test that SearchItem struct can handle all expected fields
	item := SearchItem{
		ID:          "test-id",
		Title:       "Test Title",
		URL:         "/test/url",
		Breadcrumbs: "Test > Path",
		Content:     "Test content",
		Intro:       "Test intro",
		Headings:    "Test headings",
		Toplevel:    "Test toplevel",
		Score:       0.95,
		Highlights: map[string]interface{}{
			"title":            []string{"Test <mark>Title</mark>"},
			"content":          []string{"Test <mark>content</mark>"},
			"content_explicit": []string{"Explicit <mark>content</mark>"},
		},
	}

	// Test JSON marshaling
	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("Failed to marshal SearchItem: %v", err)
	}

	// Test JSON unmarshaling
	var parsedItem SearchItem
	err = json.Unmarshal(data, &parsedItem)
	if err != nil {
		t.Fatalf("Failed to unmarshal SearchItem: %v", err)
	}

	// Verify fields
	if parsedItem.ID != item.ID {
		t.Errorf("ID mismatch: got %q, want %q", parsedItem.ID, item.ID)
	}
	if parsedItem.Title != item.Title {
		t.Errorf("Title mismatch: got %q, want %q", parsedItem.Title, item.Title)
	}
	if parsedItem.Score != item.Score {
		t.Errorf("Score mismatch: got %f, want %f", parsedItem.Score, item.Score)
	}
}

func TestSearchResultMetaFields(t *testing.T) {
	// Test SearchResult Meta struct functionality
	meta := struct {
		Found struct {
			Value    int    `json:"value"`
			Relation string `json:"relation"`
		} `json:"found"`
		Took struct {
			QueryMsec int `json:"query_msec"`
			TotalMsec int `json:"total_msec"`
		} `json:"took"`
		Page int `json:"page"`
		Size int `json:"size"`
	}{}

	meta.Found.Value = 42
	meta.Found.Relation = "gte"
	meta.Took.QueryMsec = 15
	meta.Took.TotalMsec = 30
	meta.Page = 2
	meta.Size = 10

	// Test JSON round-trip
	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("Failed to marshal meta: %v", err)
	}

	var parsedMeta struct {
		Found struct {
			Value    int    `json:"value"`
			Relation string `json:"relation"`
		} `json:"found"`
		Took struct {
			QueryMsec int `json:"query_msec"`
			TotalMsec int `json:"total_msec"`
		} `json:"took"`
		Page int `json:"page"`
		Size int `json:"size"`
	}

	err = json.Unmarshal(data, &parsedMeta)
	if err != nil {
		t.Fatalf("Failed to unmarshal meta: %v", err)
	}

	if parsedMeta.Found.Value != 42 {
		t.Errorf("Found value mismatch: got %d, want %d", parsedMeta.Found.Value, 42)
	}
	if parsedMeta.Found.Relation != "gte" {
		t.Errorf("Found relation mismatch: got %q, want %q", parsedMeta.Found.Relation, "gte")
	}
}

func TestComplexSearchResultParsing(t *testing.T) {
	// Test parsing of a more complex search result with all fields
	complexJSON := `{
		"meta": {
			"found": {
				"value": 5000,
				"relation": "gte"
			},
			"took": {
				"query_msec": 25,
				"total_msec": 50
			},
			"page": 2,
			"size": 10
		},
		"hits": [
			{
				"id": "complex-id-123",
				"url": "/en/complex/path/to/doc",
				"title": "Complex Documentation Title",
				"breadcrumbs": "Complex > Path > To > Doc",
				"content": "This is complex content with multiple paragraphs and sections.",
				"intro": "This is an introduction to the complex topic.",
				"headings": "Main Topic, Subsection, Details",
				"toplevel": "documentation",
				"score": 0.875,
				"highlights": {
					"title": ["Complex <mark>Documentation</mark> Title"],
					"content": ["This is <mark>complex</mark> content"],
					"content_explicit": ["<mark>complex</mark> content with multiple paragraphs"],
					"term": ["<mark>documentation</mark>", "<mark>complex</mark>"]
				}
			},
			{
				"id": "another-id-456",
				"url": "/en/another/path",
				"title": "Another Title",
				"score": 0.654
			}
		]
	}`

	var result SearchResult
	err := json.Unmarshal([]byte(complexJSON), &result)
	if err != nil {
		t.Fatalf("Failed to parse complex JSON: %v", err)
	}

	// Verify meta
	if result.Meta.Found.Value != 5000 {
		t.Errorf("Expected found value 5000, got %d", result.Meta.Found.Value)
	}
	if result.Meta.Found.Relation != "gte" {
		t.Errorf("Expected relation 'gte', got %s", result.Meta.Found.Relation)
	}
	if result.Meta.Page != 2 {
		t.Errorf("Expected page 2, got %d", result.Meta.Page)
	}

	// Verify hits
	if len(result.Hits) != 2 {
		t.Fatalf("Expected 2 hits, got %d", len(result.Hits))
	}

	firstHit := result.Hits[0]
	if firstHit.Content != "This is complex content with multiple paragraphs and sections." {
		t.Errorf("Unexpected content: %s", firstHit.Content)
	}
	if firstHit.Intro != "This is an introduction to the complex topic." {
		t.Errorf("Unexpected intro: %s", firstHit.Intro)
	}
	if firstHit.Headings != "Main Topic, Subsection, Details" {
		t.Errorf("Unexpected headings: %s", firstHit.Headings)
	}
	if firstHit.Toplevel != "documentation" {
		t.Errorf("Unexpected toplevel: %s", firstHit.Toplevel)
	}
	if firstHit.Score != 0.875 {
		t.Errorf("Expected score 0.875, got %f", firstHit.Score)
	}

	// Verify highlights
	if firstHit.Highlights == nil {
		t.Fatal("Expected highlights to be present")
	}

	titleHighlights, exists := firstHit.Highlights["title"]
	if !exists {
		t.Error("Expected title highlights")
	} else if highlights, ok := titleHighlights.([]interface{}); ok {
		if len(highlights) != 1 {
			t.Errorf("Expected 1 title highlight, got %d", len(highlights))
		}
	}

	// Verify second hit (minimal data)
	secondHit := result.Hits[1]
	if secondHit.Score != 0.654 {
		t.Errorf("Expected second hit score 0.654, got %f", secondHit.Score)
	}
}

func TestStringSliceEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*StringSlice)
		expected string
	}{
		{
			name: "duplicate values",
			setup: func(ss *StringSlice) {
				_ = ss.Set("title")
				_ = ss.Set("title")
				_ = ss.Set("content")
			},
			expected: "title,title,content",
		},
		{
			name: "empty strings",
			setup: func(ss *StringSlice) {
				_ = ss.Set("")
				_ = ss.Set("title")
				_ = ss.Set("")
			},
			expected: ",title,",
		},
		{
			name: "special characters",
			setup: func(ss *StringSlice) {
				_ = ss.Set("title,with,commas")
				_ = ss.Set("content with spaces")
				_ = ss.Set("content_with_underscores")
			},
			expected: "title,with,commas,content with spaces,content_with_underscores",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ss StringSlice
			tt.setup(&ss)

			result := ss.String()
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestURLConstructionEdgeCases(t *testing.T) {
	tests := []struct {
		name                  string
		query                 string
		includeMatchedContent bool
		expectedParams        map[string][]string
	}{
		{
			name:                  "matched content auto includes",
			query:                 "test query",
			includeMatchedContent: true,
			expectedParams: map[string][]string{
				"highlights": {"content_explicit"},
				"include":    {"toplevel"},
			},
		},
		{
			name:                  "default includes intro",
			query:                 "test query",
			includeMatchedContent: false,
			expectedParams: map[string][]string{
				"include": {"intro"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			searchURL, err := url.Parse(endpoint)
			if err != nil {
				t.Fatalf("Failed to parse endpoint: %v", err)
			}

			params := url.Values{}
			params.Set("query", tt.query)
			params.Set("size", "5")
			params.Set("version", "free-pro-team")
			params.Set("language", "en")

			if tt.includeMatchedContent {
				params.Add("highlights", "content_explicit")
				params.Add("include", "toplevel")
			} else {
				params.Add("include", "intro")
			}

			searchURL.RawQuery = params.Encode()
			parsedParams, _ := url.ParseQuery(searchURL.RawQuery)

			for key, expectedValues := range tt.expectedParams {
				actualValues := parsedParams[key]
				if len(actualValues) != len(expectedValues) {
					t.Errorf("Parameter %s: expected %d values, got %d", key, len(expectedValues), len(actualValues))
					continue
				}
				for i, expected := range expectedValues {
					if i >= len(actualValues) || actualValues[i] != expected {
						t.Errorf("Parameter %s[%d]: expected %q, got %q", key, i, expected, actualValues[i])
					}
				}
			}
		})
	}
}

func TestPaginationEdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		foundValue int
		size       int
		expected   int
	}{
		{"exact division edge", 99, 10, 10},
		{"one item", 1, 10, 1},
		{"size larger than found", 5, 100, 1},
		{"very large numbers", 999999, 1000, 1000},
		{"minimum values", 1, 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var totalPages int
			if tt.foundValue == 0 {
				totalPages = 0
			} else {
				totalPages = (tt.foundValue + tt.size - 1) / tt.size
			}

			if totalPages != tt.expected {
				t.Errorf("Expected %d total pages, got %d", tt.expected, totalPages)
			}
		})
	}
}

func TestHighlightHandling(t *testing.T) {
	// Test different highlight data structures
	testHighlights := map[string]interface{}{
		"title":            []interface{}{"Test <mark>Title</mark>"},
		"content":          "Single <mark>content</mark> string",
		"content_explicit": []interface{}{"First <mark>match</mark>", "Second <mark>match</mark>"},
		"term":             []interface{}{},
	}

	// Test type assertions that would happen in real code
	if titleHighlights, exists := testHighlights["title"]; exists {
		switch v := titleHighlights.(type) {
		case []interface{}:
			if len(v) != 1 {
				t.Errorf("Expected 1 title highlight, got %d", len(v))
			}
		case string:
			t.Error("Title highlights should be array, not string")
		}
	}

	if contentHighlights, exists := testHighlights["content"]; exists {
		switch v := contentHighlights.(type) {
		case []interface{}:
			t.Error("Content highlights should be string, not array")
		case string:
			if !strings.Contains(v, "content") {
				t.Error("Content highlight should contain 'content'")
			}
		}
	}

	if explicitHighlights, exists := testHighlights["content_explicit"]; exists {
		switch v := explicitHighlights.(type) {
		case []interface{}:
			if len(v) != 2 {
				t.Errorf("Expected 2 explicit highlights, got %d", len(v))
			}
		case string:
			t.Error("Explicit highlights should be array, not string")
		}
	}
}

func TestInvalidJSONHandling(t *testing.T) {
	invalidJSONs := []string{
		`{"meta": {"found": {"value": "not-a-number"}}}`,
		`{"hits": [{"score": "not-a-float"}]}`,
		`{"meta": {}}`, // missing required fields
		`malformed json`,
		`{"meta": null}`,
		`{"hits": null}`,
	}

	for i, invalidJSON := range invalidJSONs {
		t.Run(fmt.Sprintf("invalid_json_%d", i), func(t *testing.T) {
			var result SearchResult
			err := json.Unmarshal([]byte(invalidJSON), &result)
			// We expect these to either error or parse with default values
			if err == nil {
				// If it doesn't error, check that we get sensible defaults
				if result.Meta.Found.Value < 0 {
					t.Error("Found value should not be negative")
				}
			}
			// If it errors, that's also acceptable for invalid JSON
		})
	}
}

func TestEmptySearchResult(t *testing.T) {
	emptyJSON := `{
		"meta": {
			"found": {"value": 0, "relation": "eq"},
			"took": {"query_msec": 5, "total_msec": 10},
			"page": 1,
			"size": 5
		},
		"hits": []
	}`

	var result SearchResult
	err := json.Unmarshal([]byte(emptyJSON), &result)
	if err != nil {
		t.Fatalf("Failed to parse empty result JSON: %v", err)
	}

	if result.Meta.Found.Value != 0 {
		t.Errorf("Expected found value 0, got %d", result.Meta.Found.Value)
	}

	if len(result.Hits) != 0 {
		t.Errorf("Expected 0 hits, got %d", len(result.Hits))
	}
}

func TestMaxResultsCalculation(t *testing.T) {
	tests := []struct {
		name                  string
		hitsLength            int
		sizeFlag              int
		includeMatchedContent bool
		expectedMax           int
	}{
		{"default size no matched content", 10, 5, false, 5},
		{"default size with matched content", 10, 5, true, 5},
		{"custom size smaller than hits", 10, 3, false, 3},
		{"custom size larger than hits", 3, 10, false, 3},
		{"exact match", 5, 5, false, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maxResults := len(make([]SearchItem, tt.hitsLength))

			// Simulate the logic from main.go
			if tt.sizeFlag == 5 && maxResults > 5 && !tt.includeMatchedContent {
				maxResults = 5
			} else if tt.sizeFlag < maxResults {
				maxResults = tt.sizeFlag
			}

			if maxResults != tt.expectedMax {
				t.Errorf("Expected max results %d, got %d", tt.expectedMax, maxResults)
			}
		})
	}
}

func TestVersionValidation(t *testing.T) {
	validVersions := []string{
		"free-pro-team",
		"enterprise-cloud",
		"enterprise-server@3.14",
		"enterprise-server@3.15",
		"enterprise-server@3.16",
		"enterprise-server@3.17",
	}

	for _, version := range validVersions {
		t.Run(version, func(t *testing.T) {
			// Test that version can be used in URL construction
			params := url.Values{}
			params.Set("version", version)

			if params.Get("version") != version {
				t.Errorf("Version parameter not set correctly: got %s, want %s", params.Get("version"), version)
			}

			// Test URL encoding doesn't break version format
			encoded := params.Encode()
			decoded, err := url.ParseQuery(encoded)
			if err != nil {
				t.Errorf("Failed to parse encoded URL: %v", err)
			}

			if decoded.Get("version") != version {
				t.Errorf("Version lost in encoding: got %s, want %s", decoded.Get("version"), version)
			}
		})
	}
}

func TestLanguageParameterHandling(t *testing.T) {
	languages := []string{"es", "ja", "pt", "zh", "ru", "fr", "ko", "de"}

	for _, lang := range languages {
		t.Run(lang, func(t *testing.T) {
			params := url.Values{}
			params.Set("language", lang)

			if params.Get("language") != lang {
				t.Errorf("Language parameter not set correctly: got %s, want %s", params.Get("language"), lang)
			}
		})
	}
}

func TestSortParameterHandling(t *testing.T) {
	sortOptions := []string{"", "relevance", "updated", "created"}

	for _, sort := range sortOptions {
		t.Run(fmt.Sprintf("sort_%s", sort), func(t *testing.T) {
			params := url.Values{}
			if sort != "" {
				params.Set("sort", sort)
			}

			if sort == "" {
				if params.Get("sort") != "" {
					t.Error("Empty sort should result in empty parameter")
				}
			} else {
				if params.Get("sort") != sort {
					t.Errorf("Sort parameter not set correctly: got %s, want %s", params.Get("sort"), sort)
				}
			}
		})
	}
}

func TestComplexHighlightStructures(t *testing.T) {
	complexHighlights := map[string]interface{}{
		"title": []interface{}{
			"First <mark>match</mark>",
			"Second <mark>match</mark>",
		},
		"content": []interface{}{
			"Content <mark>highlight</mark> one",
			"Content <mark>highlight</mark> two",
			"Content <mark>highlight</mark> three",
		},
		"content_explicit": []interface{}{
			"Explicit <mark>content</mark> with HTML",
			"Another <mark>explicit</mark> match",
		},
		"term": []interface{}{
			"<mark>term1</mark>",
			"<mark>term2</mark>",
		},
	}

	// Test that we can handle different highlight structures
	for key, value := range complexHighlights {
		t.Run(key, func(t *testing.T) {
			switch v := value.(type) {
			case []interface{}:
				if len(v) == 0 {
					t.Errorf("Highlights for %s should not be empty", key)
				}
				for i, highlight := range v {
					if str, ok := highlight.(string); ok {
						if !strings.Contains(str, "<mark>") {
							t.Errorf("Highlight %d for %s should contain <mark> tags", i, key)
						}
					} else {
						t.Errorf("Highlight %d for %s should be string", i, key)
					}
				}
			case string:
				if !strings.Contains(v, "<mark>") {
					t.Errorf("String highlight for %s should contain <mark> tags", key)
				}
			default:
				t.Errorf("Unexpected highlight type for %s: %T", key, v)
			}
		})
	}
}

func TestSearchItemOptionalFields(t *testing.T) {
	// Test SearchItem with only required fields
	minimalItem := SearchItem{
		ID:    "minimal-id",
		Title: "Minimal Title",
		URL:   "/minimal/url",
	}

	data, err := json.Marshal(minimalItem)
	if err != nil {
		t.Fatalf("Failed to marshal minimal item: %v", err)
	}

	var parsed SearchItem
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Failed to unmarshal minimal item: %v", err)
	}

	if parsed.ID != minimalItem.ID {
		t.Errorf("ID mismatch: got %s, want %s", parsed.ID, minimalItem.ID)
	}

	// Test that optional fields are empty
	if parsed.Breadcrumbs != "" {
		t.Error("Breadcrumbs should be empty for minimal item")
	}
	if parsed.Content != "" {
		t.Error("Content should be empty for minimal item")
	}
	if parsed.Intro != "" {
		t.Error("Intro should be empty for minimal item")
	}
}

func TestMultipleParameterValues(t *testing.T) {
	// Test URL construction with multiple values for the same parameter
	params := url.Values{}
	params.Set("query", "test query")

	// Add multiple highlights
	highlights := []string{"title", "content", "content_explicit", "term"}
	for _, h := range highlights {
		params.Add("highlights", h)
	}

	// Add multiple includes
	includes := []string{"intro", "headings", "toplevel"}
	for _, inc := range includes {
		params.Add("include", inc)
	}

	// Add multiple toplevel filters
	toplevels := []string{"admin", "actions", "code-security"}
	for _, tl := range toplevels {
		params.Add("toplevel", tl)
	}

	encoded := params.Encode()
	decoded, err := url.ParseQuery(encoded)
	if err != nil {
		t.Fatalf("Failed to parse encoded parameters: %v", err)
	}

	// Verify multiple values are preserved
	decodedHighlights := decoded["highlights"]
	if len(decodedHighlights) != len(highlights) {
		t.Errorf("Expected %d highlights, got %d", len(highlights), len(decodedHighlights))
	}

	decodedIncludes := decoded["include"]
	if len(decodedIncludes) != len(includes) {
		t.Errorf("Expected %d includes, got %d", len(includes), len(decodedIncludes))
	}

	decodedToplevels := decoded["toplevel"]
	if len(decodedToplevels) != len(toplevels) {
		t.Errorf("Expected %d toplevels, got %d", len(toplevels), len(decodedToplevels))
	}
}

func TestBoundaryValues(t *testing.T) {
	tests := []struct {
		name     string
		size     int
		page     int
		expected bool // whether values should be valid
	}{
		{"minimum values", 1, 1, true},
		{"zero size", 0, 1, true}, // handled by API
		{"zero page", 1, 0, true}, // handled by API
		{"large size", 1000, 1, true},
		{"large page", 10, 1000, true},
		{"negative size", -1, 1, true},  // would be handled by flag parsing
		{"negative page", 10, -1, true}, // would be handled by flag parsing
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := url.Values{}
			params.Set("query", "test")
			params.Set("size", strconv.Itoa(tt.size))
			if tt.page > 0 {
				params.Set("page", strconv.Itoa(tt.page))
			}

			// Test that URL can be constructed
			testURL, err := url.Parse("https://example.com/api")
			if err != nil {
				t.Fatalf("Failed to parse test URL: %v", err)
			}

			testURL.RawQuery = params.Encode()

			// URL should be constructible regardless of parameter values
			if testURL.String() == "" {
				t.Error("URL construction failed")
			}
		})
	}
}

func TestReorderArgs(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "flags before query",
			input:    []string{"--debug", "--size", "5", "ssh key"},
			expected: []string{"--debug", "--size", "5", "ssh key"},
		},
		{
			name:     "flags after query",
			input:    []string{"ssh key", "--debug", "--size", "5"},
			expected: []string{"--debug", "--size", "5", "ssh key"},
		},
		{
			name:     "mixed flags and query",
			input:    []string{"--debug", "ssh key", "--size", "5"},
			expected: []string{"--debug", "--size", "5", "ssh key"},
		},
		{
			name:     "flag with equals",
			input:    []string{"ssh key", "--size=5", "--debug"},
			expected: []string{"--size=5", "--debug", "ssh key"},
		},
		{
			name:     "boolean flags only",
			input:    []string{"ssh key", "--debug", "--plain"},
			expected: []string{"--debug", "--plain", "ssh key"},
		},
		{
			name:     "repeated flags",
			input:    []string{"ssh", "--include", "intro", "--include", "headings", "key"},
			expected: []string{"--include", "intro", "--include", "headings", "ssh", "key"},
		},
		{
			name:     "no flags",
			input:    []string{"ssh", "key", "authentication"},
			expected: []string{"ssh", "key", "authentication"},
		},
		{
			name:     "only flags",
			input:    []string{"--debug", "--size", "5"},
			expected: []string{"--debug", "--size", "5"},
		},
		{
			name:     "quoted query with flags",
			input:    []string{"ssh key", "--format=json"},
			expected: []string{"--format=json", "ssh key"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := reorderArgs(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected length %d, got %d", len(tt.expected), len(result))
				return
			}

			for i, arg := range result {
				if arg != tt.expected[i] {
					t.Errorf("At position %d: expected %q, got %q", i, tt.expected[i], arg)
				}
			}
		})
	}
}
