package diagram

import (
	"fmt"
	"testing"

	"github.com/junkd0g/sharingan/internal/analyzer"
)

func TestGenerateDOT(t *testing.T) {
	arch, err := analyzer.Analyze("/Users/iordanispaschalidis/gear/offsidecompass/ai-assistant")
	if err != nil {
		t.Fatalf("Failed to analyze: %v", err)
	}

	fmt.Printf("Found %d components:\n", len(arch.Components))
	for _, comp := range arch.Components {
		fmt.Printf("  - %s (%s) in %s, deps: %v\n", comp.Name, comp.Type, comp.Package, comp.Dependencies)
	}

	dot := GenerateDOT(arch)
	fmt.Println("\nDOT output:")
	fmt.Println(dot)
}

func TestGenerate(t *testing.T) {
	arch, err := analyzer.Analyze("/Users/iordanispaschalidis/gear/offsidecompass/ai-assistant")
	if err != nil {
		t.Fatalf("Failed to analyze: %v", err)
	}

	if len(arch.Components) == 0 {
		t.Fatal("No components found")
	}

	err = Generate(arch, "/tmp/test_architecture.png")
	if err != nil {
		t.Fatalf("Failed to generate: %v", err)
	}
	t.Logf("Generated /tmp/test_architecture.png with %d components", len(arch.Components))
}
