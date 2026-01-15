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
	analyzer.ComponentHandler:    "#4A90D9", // Blue
	analyzer.ComponentService:    "#50C878", // Green
	analyzer.ComponentRepository: "#FFB347", // Orange
	analyzer.ComponentModel:      "#DDA0DD", // Plum
	analyzer.ComponentMiddleware: "#87CEEB", // Sky Blue
	analyzer.ComponentConfig:     "#D3D3D3", // Light Gray
	analyzer.ComponentUnknown:    "#FFFFFF", // White
}

// Generate creates a diagram from the architecture and saves it to the output path.
// Supports .svg, .png output formats based on the file extension.
func Generate(arch *analyzer.Architecture, outputPath string) error {
	ctx := context.Background()

	g, err := graphviz.New(ctx)
	if err != nil {
		return fmt.Errorf("failed to create graphviz: %w", err)
	}
	defer g.Close()

	graph, err := g.Graph()
	if err != nil {
		return fmt.Errorf("failed to create graph: %w", err)
	}
	defer graph.Close()

	// Set graph attributes
	graph.SetRankDir("TB")
	graph.SetLabel("Service Architecture")

	// Create nodes for each component
	nodes := make(map[string]*graphviz.Node)
	for _, comp := range arch.Components {
		node, err := graph.CreateNodeByName(sanitizeName(comp.Name))
		if err != nil {
			continue
		}

		node.SetShape("box")
		node.SetStyle("filled")
		node.SetFillColor(ColorScheme[comp.Type])
		node.SetLabel(formatLabel(comp))

		nodes[comp.Name] = node
	}

	// Create edges for dependencies
	for _, comp := range arch.Components {
		srcNode, ok := nodes[comp.Name]
		if !ok {
			continue
		}

		for _, dep := range comp.Dependencies {
			dstNode, ok := nodes[dep]
			if !ok {
				continue
			}

			edgeName := fmt.Sprintf("%s_%s", sanitizeName(comp.Name), sanitizeName(dep))
			edge, err := graph.CreateEdgeByName(edgeName, srcNode, dstNode)
			if err != nil {
				continue
			}
			edge.SetColor("#666666")
		}
	}

	// Determine format from output path
	format := graphviz.PNG
	if strings.HasSuffix(outputPath, ".svg") {
		format = graphviz.SVG
	}

	// Render to buffer
	var buf bytes.Buffer
	if err := g.Render(ctx, graph, format, &buf); err != nil {
		return fmt.Errorf("failed to render graph: %w", err)
	}

	// Write to file
	if err := writeFileBytes(outputPath, buf.Bytes()); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	return nil
}

// GenerateDOT creates a DOT representation of the architecture.
func GenerateDOT(arch *analyzer.Architecture) string {
	var sb strings.Builder

	sb.WriteString("digraph Architecture {\n")
	sb.WriteString("  rankdir=TB;\n")
	sb.WriteString("  label=\"Service Architecture\";\n")
	sb.WriteString("  labelloc=t;\n")
	sb.WriteString("  fontsize=20;\n")
	sb.WriteString("  pad=0.5;\n")
	sb.WriteString("  nodesep=0.5;\n")
	sb.WriteString("  ranksep=1.0;\n\n")

	// Group components by type
	groups := make(map[analyzer.ComponentType][]analyzer.Component)
	for _, comp := range arch.Components {
		groups[comp.Type] = append(groups[comp.Type], comp)
	}

	// Create subgraphs
	subgraphLabels := map[analyzer.ComponentType]string{
		analyzer.ComponentHandler:    "Handlers",
		analyzer.ComponentService:    "Services",
		analyzer.ComponentRepository: "Repositories",
		analyzer.ComponentModel:      "Models",
		analyzer.ComponentMiddleware: "Middleware",
		analyzer.ComponentConfig:     "Config",
	}

	for compType, label := range subgraphLabels {
		components, ok := groups[compType]
		if !ok || len(components) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("  subgraph cluster_%s {\n", compType))
		sb.WriteString(fmt.Sprintf("    label=\"%s\";\n", label))
		sb.WriteString("    style=rounded;\n")
		sb.WriteString("    bgcolor=\"#F5F5F5\";\n\n")

		for _, comp := range components {
			sb.WriteString(fmt.Sprintf("    %s [shape=box, style=filled, fillcolor=\"%s\", label=\"%s\\n(%s)\"];\n",
				sanitizeName(comp.Name), ColorScheme[compType], comp.Name, comp.Package))
		}

		sb.WriteString("  }\n\n")
	}

	// Create edges
	for _, comp := range arch.Components {
		for _, dep := range comp.Dependencies {
			sb.WriteString(fmt.Sprintf("  %s -> %s [color=\"#666666\"];\n",
				sanitizeName(comp.Name), sanitizeName(dep)))
		}
	}

	sb.WriteString("}\n")

	return sb.String()
}

func formatLabel(comp analyzer.Component) string {
	return fmt.Sprintf("%s\n(%s)", comp.Name, comp.Package)
}

func sanitizeName(name string) string {
	return strings.ReplaceAll(name, "-", "_")
}
