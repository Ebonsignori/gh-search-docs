// Command gh-search-docs is a GitHub CLI extension that searches the GitHub documentation
// using the provided docs.github.com Search API.
//
// Build / install:
//
//	gh extension install <your-username>/gh-search-docs
//
// Usage:
//
//	gh search-docs [flags] <query>
//
// Flags:
//
//	--size        number of results to return (max: 50, default: 5)
//	--version     docs version (free-pro-team, enterprise-cloud,
//	              or enterprise-server@<3.13-3.17>)
//	--language    language code (default: en)
//	--page        page number for pagination
//	--sort        sort order
//	--highlights           highlight options: title, content, content_explicit, term
//	--include              additional includes: intro, headings, toplevel
//	--include-matched-content include matched content highlights
//	--toplevel             toplevel filter
//	--aggregate            aggregate options
//	--debug                show raw JSON response from the API
//	--format               output format: pretty (default), plain, json
//	--plain                disable pretty rendering (use plain text output)
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/glamour"

	"github.com/Ebonsignori/gh-search-docs/searchdocs"
)

const endpoint = "https://docs.github.com/api/search/v1"

type SearchResult struct {
	Meta struct {
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
	} `json:"meta"`
	Hits []SearchItem `json:"hits"`
}

type SearchItem struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	URL         string                 `json:"url"`
	Breadcrumbs string                 `json:"breadcrumbs,omitempty"`
	Content     string                 `json:"content,omitempty"`
	Intro       string                 `json:"intro,omitempty"`
	Headings    string                 `json:"headings,omitempty"`
	Toplevel    string                 `json:"toplevel,omitempty"`
	Highlights  map[string]interface{} `json:"highlights,omitempty"`
	Score       float64                `json:"score,omitempty"`
}

// StringSlice allows repeated flags
type StringSlice []string

func (s *StringSlice) String() string {
	return strings.Join(*s, ",")
}

func (s *StringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

// reorderArgs separates flags from non-flag arguments and returns them with flags first.
// This allows flags to be specified after the query (e.g., "query" --debug).
func reorderArgs(args []string) []string {
	var flags []string
	var nonFlags []string

	// Boolean flags that don't take values
	boolFlags := map[string]bool{
		"--debug":                   true,
		"--plain":                   true,
		"--list-versions":           true,
		"--include-matched-content": true,
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			// Check if flag contains '=' (e.g., --size=5)
			if strings.Contains(arg, "=") {
				// Flag with embedded value, add as-is
				flags = append(flags, arg)
			} else if boolFlags[arg] {
				// Boolean flag, no value expected
				flags = append(flags, arg)
			} else {
				// Flag that expects a value
				flags = append(flags, arg)
				// Check if the next argument exists and doesn't start with "-"
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					// Include the flag's value
					i++
					flags = append(flags, args[i])
				}
			}
		} else {
			// This is a non-flag argument (part of the query)
			nonFlags = append(nonFlags, arg)
		}
	}

	// Return flags first, then non-flag arguments
	return append(flags, nonFlags...)
}

func main() {
	//----------------------------------------------------------------------
	// Flags
	//----------------------------------------------------------------------
	fs := flag.NewFlagSet("search-docs", flag.ExitOnError)

	queryFlag := fs.String("query", "", "search query (can also be provided as positional argument)")
	sizeFlag := fs.Int("size", 5, "number of results to return (max: 50, default shows top 5 with links and descriptions)")
	versionFlag := fs.String("version", "free-pro-team", "docs version")
	languageFlag := fs.String("language", "en", "language code")
	pageFlag := fs.Int("page", 0, "page number for pagination")
	sortFlag := fs.String("sort", "", "sort order")
	debugFlag := fs.Bool("debug", false, "show raw JSON response")
	formatFlag := fs.String("format", "pretty", "output format: pretty (default), plain, json")
	plainFlag := fs.Bool("plain", false, "disable pretty rendering (use plain text output)")
	listVersions := fs.Bool("list-versions", false, "list supported enterprise server versions")
	includeMatchedContentFlag := fs.Bool("include-matched-content", false, "include matched content highlights")

	var highlights StringSlice
	var includes StringSlice
	var toplevel StringSlice
	var aggregate StringSlice

	fs.Var(&highlights, "highlights", "highlight options (can be used multiple times): title, content, content_explicit, term")
	fs.Var(&includes, "include", "additional includes (can be used multiple times): intro, headings, toplevel")
	fs.Var(&toplevel, "toplevel", "toplevel filter (can be used multiple times)")
	fs.Var(&aggregate, "aggregate", "aggregate options (can be used multiple times)")

	fs.Usage = func() {
		bin := filepath.Base(os.Args[0])
		if strings.HasPrefix(bin, "gh-") {
			bin = "gh " + strings.TrimPrefix(bin, "gh-")
		}
		fmt.Fprintf(os.Stderr, "usage: %s [flags] <query>\n\n", bin)
		fmt.Fprintf(os.Stderr, "By default, output uses pretty formatting with colors.\n")
		fmt.Fprintf(os.Stderr, "Use --plain for simple text output with clickable URLs.\n\n")
		fs.PrintDefaults()
	}

	// Reorder arguments to allow flags after the query
	reorderedArgs := reorderArgs(os.Args[1:])

	if err := fs.Parse(reorderedArgs); err != nil {
		searchdocs.Fatal(err)
	}

	if *listVersions {
		versions, err := searchdocs.LoadSupportedVersions()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading supported versions: %v\n", err)
			fmt.Fprintf(os.Stderr, "Fallback supported versions: 3.11, 3.12, 3.13, 3.14, 3.15, 3.16, 3.17\n")
			os.Exit(1)
		}

		fmt.Println("Supported GitHub Enterprise Server versions:")
		for _, version := range versions.SupportedVersions {
			if version == versions.LatestVersion {
				fmt.Printf("  %s (latest)\n", version)
			} else {
				fmt.Printf("  %s\n", version)
			}
		}
		fmt.Printf("\nLast updated: %s\n", versions.LastUpdated)
		fmt.Println("\nUsage: gh search-docs --version enterprise-server@<version> <query>")
		os.Exit(0)
	}

	// Get query from flag or positional arguments
	query := *queryFlag
	if query == "" && fs.NArg() > 0 {
		query = strings.Join(fs.Args(), " ")
	}

	if query == "" {
		fs.Usage()
		os.Exit(1)
	}

	// Validate size flag - GitHub Docs API has a maximum limit of 50
	if *sizeFlag > 50 {
		fmt.Fprintf(os.Stderr, "Error: --size cannot exceed 50 (GitHub Docs API limit). Use --page to navigate through more results.\n")
		os.Exit(1)
	}
	if *sizeFlag < 1 {
		fmt.Fprintf(os.Stderr, "Error: --size must be at least 1.\n")
		os.Exit(1)
	}

	version := searchdocs.NormalizeVersion(*versionFlag)

	//----------------------------------------------------------------------
	// Build URL with query parameters
	//----------------------------------------------------------------------
	searchURL, err := url.Parse(endpoint)
	if err != nil {
		searchdocs.Fatal(err)
	}

	params := url.Values{}
	params.Set("query", query)
	params.Set("size", strconv.Itoa(*sizeFlag))
	params.Set("version", version)
	params.Set("language", *languageFlag)

	if *pageFlag > 0 {
		params.Set("page", strconv.Itoa(*pageFlag))
	}
	if *sortFlag != "" {
		params.Set("sort", *sortFlag)
	}
	if len(highlights) > 0 {
		for _, h := range highlights {
			params.Add("highlights", h)
		}
	}
	if *includeMatchedContentFlag {
		// Auto-add content_explicit highlights for matched content
		params.Add("highlights", "content_explicit")
	}
	// Auto-include intro for descriptions unless user specified includes
	if len(includes) == 0 {
		if *includeMatchedContentFlag {
			// For matched content, we need at least one include field for API compatibility
			params.Add("include", "toplevel")
		} else {
			// Default behavior - include intro
			params.Add("include", "intro")
		}
	} else {
		for _, inc := range includes {
			params.Add("include", inc)
		}
	}
	if len(toplevel) > 0 {
		for _, tl := range toplevel {
			params.Add("toplevel", tl)
		}
	}
	if len(aggregate) > 0 {
		for _, agg := range aggregate {
			params.Add("aggregate", agg)
		}
	}

	searchURL.RawQuery = params.Encode()

	//----------------------------------------------------------------------
	// HTTP Request
	//----------------------------------------------------------------------
	req, err := http.NewRequest(http.MethodGet, searchURL.String(), nil)
	if err != nil {
		searchdocs.Fatal(err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error making request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "API returned status %d\n", resp.StatusCode)
		if resp.StatusCode == 429 {
			fmt.Fprintf(os.Stderr, "Rate limited. Please try again later.\n")
		}
		os.Exit(1)
	}

	//----------------------------------------------------------------------
	// Parse Response
	//----------------------------------------------------------------------
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		searchdocs.Fatal(err)
	}

	if *debugFlag {
		fmt.Fprintf(os.Stderr, "Raw response:\n%s\n", body)
	}

	var result SearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
		if *debugFlag {
			fmt.Fprintf(os.Stderr, "Response body: %s\n", body)
		}
		os.Exit(1)
	}

	//----------------------------------------------------------------------
	// Output Results
	//----------------------------------------------------------------------
	if *formatFlag == "json" {
		output, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			searchdocs.Fatal(err)
		}
		fmt.Println(string(output))
		return
	}

	if result.Meta.Found.Value == 0 {
		fmt.Printf("No results found for query: %s\n", query)
		return
	}

	fmt.Printf("Found %d results", result.Meta.Found.Value)
	if result.Meta.Page > 1 {
		fmt.Printf(" (page %d)", result.Meta.Page)
	}
	fmt.Println()

	// Determine how many results to show and what level of detail
	maxResults := len(result.Hits)
	// Always respect user-specified size, but limit to 5 by default when no special flags
	if *sizeFlag == 5 && maxResults > 5 && !*includeMatchedContentFlag {
		maxResults = 5
	} else if *sizeFlag < maxResults {
		maxResults = *sizeFlag
	}

	// Check if we should use pretty rendering or plain text
	// Pretty is now the default unless explicitly disabled
	usePrettyRendering := !*plainFlag && *formatFlag != "plain"

	var renderer *glamour.TermRenderer
	if usePrettyRendering {
		// Create renderer for pretty output without word wrapping
		renderer = searchdocs.NewAutoRendererNoWrap()
		if renderer == nil {
			theme := "dark"
			if searchdocs.IsLight() {
				theme = "light"
			}
			renderer = searchdocs.NewRendererNoWrap(theme)
		}
	}

	for i := 0; i < maxResults; i++ {
		item := result.Hits[i]

		if usePrettyRendering {
			// Pretty rendering with markdown
			var md strings.Builder
			md.WriteString(fmt.Sprintf("%d. %s\n", i+1, item.Title))
			md.WriteString(fmt.Sprintf("   %s\n", "https://docs.github.com"+item.URL))

			// Show summary by default unless matched content is requested
			if !*includeMatchedContentFlag {
				if item.Intro != "" {
					description := item.Intro
					if len(description) > 150 {
						description = description[:150] + "..."
					}
					md.WriteString(fmt.Sprintf("   %s\n", description))
				}
			}

			// Show matched content if flag is set
			if *includeMatchedContentFlag && item.Highlights != nil {
				if contentExplicit, exists := item.Highlights["content_explicit"]; exists {
					switch v := contentExplicit.(type) {
					case []interface{}:
						for _, highlight := range v {
							if str, ok := highlight.(string); ok {
								md.WriteString(fmt.Sprintf("   • %s\n", str))
							}
						}
					case string:
						md.WriteString(fmt.Sprintf("   • %s\n", v))
					}
				}
			}

			md.WriteString("\n")

			// Render the markdown
			if renderer != nil {
				output, err := renderer.Render(md.String())
				if err == nil {
					fmt.Print(output)
					continue
				}
			}

			// Fallback to plain text if rendering fails
			fmt.Print(md.String())
		} else {
			// Plain text output - URLs will never be wrapped
			fmt.Printf("%d. %s\n", i+1, item.Title)
			fmt.Printf("   %s\n", "https://docs.github.com"+item.URL)

			// Show summary by default unless matched content is requested
			if !*includeMatchedContentFlag {
				if item.Intro != "" {
					description := item.Intro
					if len(description) > 150 {
						description = description[:150] + "..."
					}
					fmt.Printf("   %s\n", description)
				}
			}

			// Show matched content if flag is set
			if *includeMatchedContentFlag && item.Highlights != nil {
				if contentExplicit, exists := item.Highlights["content_explicit"]; exists {
					switch v := contentExplicit.(type) {
					case []interface{}:
						for _, highlight := range v {
							if str, ok := highlight.(string); ok {
								// Remove HTML tags for plain text output
								cleanStr := strings.ReplaceAll(str, "<mark>", "")
								cleanStr = strings.ReplaceAll(cleanStr, "</mark>", "")
								fmt.Printf("   • %s\n", cleanStr)
							}
						}
					case string:
						// Remove HTML tags for plain text output
						cleanStr := strings.ReplaceAll(v, "<mark>", "")
						cleanStr = strings.ReplaceAll(cleanStr, "</mark>", "")
						fmt.Printf("   • %s\n", cleanStr)
					}
				}
			}

			fmt.Println()
		}
	}

	// Show info about remaining results if there are more than shown
	if maxResults == 5 && result.Meta.Found.Value > 5 && !*includeMatchedContentFlag {
		if result.Meta.Found.Value <= 50 {
			fmt.Printf("Showing top 5 results. Use --size %d to see all %d results.\n", result.Meta.Found.Value, result.Meta.Found.Value)
		} else {
			fmt.Printf("Showing top 5 results. Use --size 50 to see the maximum 50 results per page.\n")
			fmt.Printf("Use --page to navigate through all %d results.\n", result.Meta.Found.Value)
		}
		fmt.Printf("Use --include-matched-content for highlighted matches instead of descriptions.\n\n")
	}

	// Show pagination info
	totalPages := (result.Meta.Found.Value + result.Meta.Size - 1) / result.Meta.Size
	if totalPages > 1 {
		fmt.Printf("\nShowing page %d of %d (%d total results)\n",
			result.Meta.Page,
			totalPages,
			result.Meta.Found.Value)

		if result.Meta.Page < totalPages {
			fmt.Printf("Use --page %d to see the next page\n", result.Meta.Page+1)
		}
	}
}
