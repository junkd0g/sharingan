package diagram

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/goccy/go-graphviz"
	"github.com/junkd0g/sharingan/internal/analyzer"
)

// ColorScheme defines colors for different component types.
var ColorScheme = map[analyzer.ComponentType]string{
	analyzer.ComponentHandler:    "#4A90D9", // Blue - HTTP/Transport layer
	analyzer.ComponentService:    "#50C878", // Green - Business logic
	analyzer.ComponentRepository: "#FFB347", // Orange - Data access
	analyzer.ComponentAdapter:    "#9B59B6", // Purple - External integrations
}

// LayerLabels maps component types to display labels.
var LayerLabels = map[analyzer.ComponentType]string{
	analyzer.ComponentHandler:    "Transport Layer",
	analyzer.ComponentService:    "Service Layer",
	analyzer.ComponentRepository: "Data Layer",
	analyzer.ComponentAdapter:    "Adapters",
}

// Generate creates a diagram from the architecture and saves it to the output path.
func Generate(arch *analyzer.Architecture, outputPath string) error {
	ctx := context.Background()

	g, err := graphviz.New(ctx)
	if err != nil {
		return fmt.Errorf("failed to create graphviz: %w", err)
	}
	defer g.Close()

	dotString := GenerateDOT(arch)

	graph, err := graphviz.ParseBytes([]byte(dotString))
	if err != nil {
		return fmt.Errorf("failed to parse DOT: %w", err)
	}
	defer graph.Close()

	format := graphviz.PNG
	if strings.HasSuffix(outputPath, ".svg") {
		format = graphviz.SVG
	}

	var buf bytes.Buffer
	if err := g.Render(ctx, graph, format, &buf); err != nil {
		return fmt.Errorf("failed to render graph: %w", err)
	}

	if err := writeFileBytes(outputPath, buf.Bytes()); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	return nil
}

// GenerateDOT creates a clean, layered DOT representation of the architecture.
func GenerateDOT(arch *analyzer.Architecture) string {
	var sb strings.Builder

	sb.WriteString("digraph Architecture {\n")
	sb.WriteString("  rankdir=TB;\n")
	sb.WriteString("  label=\"Service Architecture\";\n")
	sb.WriteString("  labelloc=t;\n")
	sb.WriteString("  fontsize=24;\n")
	sb.WriteString("  fontname=\"Helvetica-Bold\";\n")
	sb.WriteString("  pad=0.5;\n")
	sb.WriteString("  nodesep=0.8;\n")
	sb.WriteString("  ranksep=1.2;\n")
	sb.WriteString("  splines=polyline;\n\n")

	// Default node style - bigger, more readable
	sb.WriteString("  node [fontname=\"Helvetica\", fontsize=14, margin=\"0.3,0.2\", penwidth=2];\n")
	sb.WriteString("  edge [fontname=\"Helvetica\", fontsize=10, penwidth=2, color=\"#555555\"];\n\n")

	// Group components by type
	groups := make(map[analyzer.ComponentType][]analyzer.Component)
	for _, comp := range arch.Components {
		groups[comp.Type] = append(groups[comp.Type], comp)
	}

	// Layer order: top to bottom
	layerOrder := []analyzer.ComponentType{
		analyzer.ComponentHandler,
		analyzer.ComponentService,
		analyzer.ComponentAdapter,
		analyzer.ComponentRepository,
	}

	// Create layered subgraphs
	for _, compType := range layerOrder {
		components, ok := groups[compType]
		if !ok || len(components) == 0 {
			continue
		}

		label := LayerLabels[compType]
		color := ColorScheme[compType]

		sb.WriteString(fmt.Sprintf("  subgraph cluster_%s {\n", compType))
		sb.WriteString(fmt.Sprintf("    label=\"%s\";\n", label))
		sb.WriteString("    style=\"rounded,filled\";\n")
		sb.WriteString("    fillcolor=\"#FAFAFA\";\n")
		sb.WriteString("    color=\"#CCCCCC\";\n")
		sb.WriteString("    fontsize=16;\n")
		sb.WriteString("    fontname=\"Helvetica-Bold\";\n")
		sb.WriteString("    margin=20;\n\n")

		for _, comp := range components {
			nodeName := sanitizeName(comp.Name)
			// Show package in label for context
			nodeLabel := comp.Name
			if comp.Package != "" && comp.Package != comp.Name {
				nodeLabel = fmt.Sprintf("%s\\n(%s)", comp.Name, comp.Package)
			}
			sb.WriteString(fmt.Sprintf("    %s [shape=box, style=\"rounded,filled\", fillcolor=\"%s\", label=\"%s\", fontcolor=\"white\"];\n",
				nodeName, color, nodeLabel))
		}

		sb.WriteString("  }\n\n")
	}

	// Create edges for dependencies
	sb.WriteString("  // Dependencies\n")
	for _, comp := range arch.Components {
		srcName := sanitizeName(comp.Name)
		for _, dep := range comp.Dependencies {
			dstName := sanitizeName(dep)
			sb.WriteString(fmt.Sprintf("  %s -> %s;\n", srcName, dstName))
		}
	}

	sb.WriteString("}\n")

	return sb.String()
}

func sanitizeName(name string) string {
	s := strings.ReplaceAll(name, "-", "_")
	s = strings.ReplaceAll(s, ".", "_")
	return s
}
