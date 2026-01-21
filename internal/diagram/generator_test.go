package diagram

import (
	"fmt"
	"os"
	"testing"

	"github.com/junkd0g/sharingan/internal/analyzer"
)

func TestGenerateHTML(t *testing.T) {
	arch, err := analyzer.Analyze("/Users/iordanispaschalidis/gear/offsidecompass/ai-assistant")
	if err != nil {
		t.Fatalf("Failed to analyze: %v", err)
	}

	fmt.Printf("Found %d components:\n", len(arch.Components))
	for _, comp := range arch.Components {
		fmt.Printf("  - %s (%s) in %s, deps: %v\n", comp.Name, comp.Type, comp.Package, comp.Dependencies)
	}

	if len(arch.Components) == 0 {
		t.Fatal("No components found")
	}

	// Test with default config
	config := DefaultConfig()
	outputPath := "/tmp/test_architecture.html"

	err = GenerateHTML(arch, outputPath, config)
	if err != nil {
		t.Fatalf("Failed to generate HTML: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("HTML file was not created")
	}

	t.Logf("Generated %s with %d components", outputPath, len(arch.Components))
}

func TestGenerateHTMLWithCustomWidgets(t *testing.T) {
	arch, err := analyzer.Analyze("/Users/iordanispaschalidis/gear/offsidecompass/ai-assistant")
	if err != nil {
		t.Fatalf("Failed to analyze: %v", err)
	}

	if len(arch.Components) == 0 {
		t.Fatal("No components found")
	}

	// Test with custom widgets
	config := HTMLConfig{
		Title:       "Custom Report",
		Description: "Testing custom widget selection",
		Theme:       "light",
		Widgets: []WidgetType{
			WidgetStatsCards,
			WidgetArchitectureGraph,
			WidgetComponentsTable,
		},
	}

	outputPath := "/tmp/test_architecture_custom.html"

	err = GenerateHTML(arch, outputPath, config)
	if err != nil {
		t.Fatalf("Failed to generate HTML: %v", err)
	}

	t.Logf("Generated %s with custom widgets", outputPath)
}

func TestGenerateHTMLLightTheme(t *testing.T) {
	arch, err := analyzer.Analyze("/Users/iordanispaschalidis/gear/offsidecompass/ai-assistant")
	if err != nil {
		t.Fatalf("Failed to analyze: %v", err)
	}

	if len(arch.Components) == 0 {
		t.Fatal("No components found")
	}

	config := DefaultConfig()
	config.Theme = "light"

	outputPath := "/tmp/test_architecture_light.html"

	err = GenerateHTML(arch, outputPath, config)
	if err != nil {
		t.Fatalf("Failed to generate HTML: %v", err)
	}

	t.Logf("Generated %s with light theme", outputPath)
}
