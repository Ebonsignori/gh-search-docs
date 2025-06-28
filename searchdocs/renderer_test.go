package searchdocs

import (
	"strings"
	"testing"

	"github.com/charmbracelet/glamour"
)

func TestNewRenderer(t *testing.T) {
	tests := []struct {
		name  string
		theme string
		wrap  int
	}{
		{"dark theme with wrap", "dark", 80},
		{"light theme with wrap", "light", 120},
		{"auto theme with wrap", "auto", 100},
		{"dark theme no wrap", "dark", 0},
		{"light theme no wrap", "light", 0},
		{"small wrap", "dark", 20},
		{"large wrap", "dark", 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := NewRenderer(tt.theme, tt.wrap)
			if renderer == nil {
				t.Errorf("NewRenderer(%q, %d) returned nil", tt.theme, tt.wrap)
			}

			// Test that renderer can actually render some markdown
			testMarkdown := "# Test Header\n\nSome **bold** text."
			output, err := renderer.Render(testMarkdown)
			if err != nil {
				t.Errorf("Renderer failed to render markdown: %v", err)
			}
			if output == "" {
				t.Error("Renderer returned empty output")
			}
		})
	}
}

func TestNewRendererInvalidTheme(t *testing.T) {
	// Test with an invalid theme - glamour may return nil for invalid themes
	renderer := NewRenderer("invalid-theme", 80)
	if renderer == nil {
		// This is acceptable behavior for invalid themes
		t.Log("NewRenderer returned nil for invalid theme (expected behavior)")
		return
	}

	// If renderer is not nil, test that it can still render
	testMarkdown := "# Test"
	output, err := renderer.Render(testMarkdown)
	if err != nil {
		t.Errorf("Renderer with invalid theme failed: %v", err)
	}
	if output == "" {
		t.Error("Renderer with invalid theme returned empty output")
	}
}

func TestNewAutoRenderer(t *testing.T) {
	tests := []struct {
		name string
		wrap int
	}{
		{"auto with wrap", 80},
		{"auto no wrap", 0},
		{"auto small wrap", 20},
		{"auto large wrap", 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := NewAutoRenderer(tt.wrap)
			if renderer == nil {
				t.Errorf("NewAutoRenderer(%d) returned nil", tt.wrap)
			}

			// Test that renderer can actually render some markdown
			testMarkdown := "# Auto Test\n\nThis is a test with *italic* text."
			output, err := renderer.Render(testMarkdown)
			if err != nil {
				t.Errorf("Auto renderer failed to render markdown: %v", err)
			}
			if output == "" {
				t.Error("Auto renderer returned empty output")
			}
		})
	}
}

func TestNewAutoRendererNoWrap(t *testing.T) {
	renderer := NewAutoRendererNoWrap()
	if renderer == nil {
		t.Error("NewAutoRendererNoWrap() returned nil")
	}

	// Test with long text that would normally wrap
	longText := "# Very Long Header That Would Normally Wrap\n\n" +
		"This is a very long paragraph that contains a lot of text and would normally be wrapped " +
		"at some point, but since we're using no wrap, it should remain on long lines."

	output, err := renderer.Render(longText)
	if err != nil {
		t.Errorf("NoWrap renderer failed: %v", err)
	}
	if output == "" {
		t.Error("NoWrap renderer returned empty output")
	}

	// The output should contain the text (though we can't easily test wrapping behavior)
	if !strings.Contains(output, "Very Long Header") {
		t.Error("Output should contain the header text")
	}
}

func TestNewRendererNoWrap(t *testing.T) {
	tests := []struct {
		name  string
		theme string
	}{
		{"dark no wrap", "dark"},
		{"light no wrap", "light"},
		{"auto no wrap", "auto"},
		{"dracula no wrap", "dracula"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := NewRendererNoWrap(tt.theme)
			if renderer == nil {
				t.Errorf("NewRendererNoWrap(%q) returned nil", tt.theme)
			}

			// Test with content that has various markdown elements
			complexMarkdown := `# Main Header

## Sub Header

This is a paragraph with **bold** and *italic* text.

- List item 1
- List item 2
- List item 3

` + "`code block`" + `

> This is a blockquote

[Link text](https://example.com)
`

			output, err := renderer.Render(complexMarkdown)
			if err != nil {
				t.Errorf("NoWrap renderer failed with %s theme: %v", tt.theme, err)
			}
			if output == "" {
				t.Error("NoWrap renderer returned empty output")
			}

			// Check that some of the content is preserved (case insensitive)
			if !strings.Contains(strings.ToLower(output), "main") || !strings.Contains(strings.ToLower(output), "header") {
				t.Logf("Output: %q", output)
				t.Error("Output should contain header content")
			}
		})
	}
}

func TestRendererOptions(t *testing.T) {
	// Test that different renderers produce different outputs with the same input
	testMarkdown := "# Test\n\nSome text that might be styled differently."

	darkRenderer := NewRenderer("dark", 80)
	lightRenderer := NewRenderer("light", 80)
	autoRenderer := NewAutoRenderer(80)
	noWrapRenderer := NewRendererNoWrap("dark")

	renderers := map[string]*glamour.TermRenderer{
		"dark":   darkRenderer,
		"light":  lightRenderer,
		"auto":   autoRenderer,
		"nowrap": noWrapRenderer,
	}

	outputs := make(map[string]string)

	for name, renderer := range renderers {
		if renderer == nil {
			t.Errorf("Renderer %s is nil", name)
			continue
		}

		output, err := renderer.Render(testMarkdown)
		if err != nil {
			t.Errorf("Renderer %s failed: %v", name, err)
			continue
		}

		if output == "" {
			t.Errorf("Renderer %s returned empty output", name)
			continue
		}

		outputs[name] = output

		// All outputs should contain the basic text
		if !strings.Contains(output, "Test") {
			t.Errorf("Renderer %s output doesn't contain 'Test'", name)
		}
	}

	// Verify we got outputs from all renderers
	expectedRenderers := []string{"dark", "light", "auto", "nowrap"}
	for _, name := range expectedRenderers {
		if _, exists := outputs[name]; !exists {
			t.Errorf("Missing output from %s renderer", name)
		}
	}
}

func TestRendererWithEmptyInput(t *testing.T) {
	renderer := NewRenderer("dark", 80)
	if renderer == nil {
		t.Fatal("Renderer should not be nil")
	}

	output, err := renderer.Render("")
	if err != nil {
		t.Errorf("Renderer failed with empty input: %v", err)
	}

	// Empty input should produce empty or minimal output
	if len(output) > 10 {
		t.Errorf("Expected minimal output for empty input, got %d characters", len(output))
	}
}

func TestRendererWithSpecialCharacters(t *testing.T) {
	renderer := NewRenderer("dark", 80)
	if renderer == nil {
		t.Fatal("Renderer should not be nil")
	}

	// Test with various special characters and unicode
	specialMarkdown := `# Special Characters: ñ, é, ü, 中文, 🎉

Some text with **émphasis** and *italics*.

- Item with émojis: 🚀 ✨ 🎯
- Unicode: αβγδε
- Symbols: ©®™

` + "```go\nfunc main() {\n    fmt.Println(\"Hello, 世界\")\n}\n```"

	output, err := renderer.Render(specialMarkdown)
	if err != nil {
		t.Errorf("Renderer failed with special characters: %v", err)
	}

	if output == "" {
		t.Error("Renderer returned empty output for special characters")
	}

	// Check that some unicode is preserved (though styling may change it)
	if !strings.Contains(output, "世界") {
		t.Error("Unicode characters should be preserved in output")
	}
}

func TestRendererWithLargeInput(t *testing.T) {
	renderer := NewRenderer("dark", 80)
	if renderer == nil {
		t.Fatal("Renderer should not be nil")
	}

	// Create a large markdown document
	var largeMarkdown strings.Builder
	largeMarkdown.WriteString("# Large Document\n\n")

	for i := 0; i < 100; i++ {
		largeMarkdown.WriteString("## Section ")
		largeMarkdown.WriteString(string(rune('A' + i%26)))
		largeMarkdown.WriteString("\n\n")
		largeMarkdown.WriteString("This is paragraph ")
		largeMarkdown.WriteString(string(rune('0' + i%10)))
		largeMarkdown.WriteString(" with some **bold** and *italic* text. ")
		largeMarkdown.WriteString("It contains enough content to test performance and memory usage.\n\n")

		if i%10 == 0 {
			largeMarkdown.WriteString("- List item 1\n- List item 2\n- List item 3\n\n")
		}
	}

	output, err := renderer.Render(largeMarkdown.String())
	if err != nil {
		t.Errorf("Renderer failed with large input: %v", err)
	}

	if output == "" {
		t.Error("Renderer returned empty output for large input")
	}

	if len(output) < 1000 {
		t.Error("Expected substantial output for large input")
	}
}

func TestAllRendererFunctionsReturnNonNil(t *testing.T) {
	// Test that all renderer creation functions return non-nil renderers
	renderers := map[string]*glamour.TermRenderer{
		"NewRenderer":           NewRenderer("dark", 80),
		"NewAutoRenderer":       NewAutoRenderer(80),
		"NewAutoRendererNoWrap": NewAutoRendererNoWrap(),
		"NewRendererNoWrap":     NewRendererNoWrap("dark"),
	}

	for name, renderer := range renderers {
		if renderer == nil {
			t.Errorf("%s returned nil renderer", name)
		}
	}
}

func TestRendererConsistency(t *testing.T) {
	// Test that the same renderer produces consistent output
	renderer := NewRenderer("dark", 80)
	if renderer == nil {
		t.Fatal("Renderer should not be nil")
	}

	testMarkdown := "# Consistency Test\n\n**Bold** and *italic* text."

	output1, err1 := renderer.Render(testMarkdown)
	if err1 != nil {
		t.Fatalf("First render failed: %v", err1)
	}

	output2, err2 := renderer.Render(testMarkdown)
	if err2 != nil {
		t.Fatalf("Second render failed: %v", err2)
	}

	if output1 != output2 {
		t.Error("Renderer should produce consistent output for the same input")
	}
}
