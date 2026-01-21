package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/junkd0g/sharingan/internal/analyzer"
	"github.com/junkd0g/sharingan/internal/diagram"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Register registers all tools with the MCP server.
func Register(s *server.MCPServer) {
	registerArchDiagramTool(s)
}

func registerArchDiagramTool(s *server.MCPServer) {
	tool := mcp.NewTool("generate_architecture_diagram",
		mcp.WithDescription(`Generates an interactive HTML architecture report from a Go service repository.

The report includes various visualizations powered by ECharts:
- Architecture Graph: Interactive force-directed graph showing components and dependencies
- Components Pie: Pie chart showing component distribution by type
- Dependencies Bar: Bar chart showing top components by dependency count
- Layer Flow: Sankey diagram showing data flow between architectural layers
- Dependency Matrix: Heatmap showing which components depend on which
- Components Table: Detailed table of all components
- Package Tree: Tree visualization of package structure
- Stats Cards: Key metrics overview

You can customize which widgets appear in the report using the 'widgets' parameter.`),
		mcp.WithString("repo_path",
			mcp.Required(),
			mcp.Description("The absolute path to the Go service repository to analyze"),
		),
		mcp.WithString("output_path",
			mcp.Description("The output path for the HTML file. Defaults to ./architecture.html in the repo"),
		),
		mcp.WithString("title",
			mcp.Description("Custom title for the report. Defaults to 'Go Architecture Report'"),
		),
		mcp.WithString("description",
			mcp.Description("Custom description shown below the title"),
		),
		mcp.WithString("theme",
			mcp.Description("Color theme: 'dark' (default) or 'light'"),
		),
		mcp.WithString("widgets",
			mcp.Description(`Comma-separated list of widgets to include. Available widgets:
- stats_cards: Key metrics cards
- architecture_graph: Interactive component graph
- components_pie: Component type distribution
- dependencies_bar: Top dependencies chart
- layer_flow: Sankey diagram of layer dependencies
- dependency_matrix: Heatmap of dependencies (max 20 components)
- components_table: Detailed component table
- package_tree: Package structure tree

Default: all widgets. Example: "stats_cards,architecture_graph,components_table"`),
		),
	)

	s.AddTool(tool, archDiagramHandler)
}

func archDiagramHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repoPath, ok := request.Params.Arguments["repo_path"].(string)
	if !ok {
		return newToolResultError("repo_path is required"), nil
	}

	// Validate repo path exists
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return newToolResultError(fmt.Sprintf("repository path does not exist: %s", repoPath)), nil
	}

	// Determine output path
	outputPath := filepath.Join(repoPath, "architecture.html")
	if op, ok := request.Params.Arguments["output_path"].(string); ok && op != "" {
		outputPath = op
	}

	// Build config
	config := diagram.DefaultConfig()

	if title, ok := request.Params.Arguments["title"].(string); ok && title != "" {
		config.Title = title
	}

	if desc, ok := request.Params.Arguments["description"].(string); ok && desc != "" {
		config.Description = desc
	}

	if theme, ok := request.Params.Arguments["theme"].(string); ok && theme != "" {
		if theme == "light" || theme == "dark" {
			config.Theme = theme
		}
	}

	if widgetsStr, ok := request.Params.Arguments["widgets"].(string); ok && widgetsStr != "" {
		config.Widgets = parseWidgets(widgetsStr)
	}

	// Analyze the repository
	arch, err := analyzer.Analyze(repoPath)
	if err != nil {
		return newToolResultError(fmt.Sprintf("failed to analyze repository: %v", err)), nil
	}

	if len(arch.Components) == 0 {
		return newToolResultError("no architectural components found in the repository"), nil
	}

	// Generate the HTML report
	if err := diagram.GenerateHTML(arch, outputPath, config); err != nil {
		return newToolResultError(fmt.Sprintf("failed to generate report: %v", err)), nil
	}

	// Build summary
	summary := buildSummary(arch, outputPath, config)

	return mcp.NewToolResultText(summary), nil
}

func parseWidgets(widgetsStr string) []diagram.WidgetType {
	widgetMap := map[string]diagram.WidgetType{
		"stats_cards":        diagram.WidgetStatsCards,
		"architecture_graph": diagram.WidgetArchitectureGraph,
		"components_pie":     diagram.WidgetComponentsPie,
		"dependencies_bar":   diagram.WidgetDependenciesBar,
		"layer_flow":         diagram.WidgetLayerFlow,
		"dependency_matrix":  diagram.WidgetDependencyMatrix,
		"components_table":   diagram.WidgetComponentsTable,
		"package_tree":       diagram.WidgetPackageTree,
	}

	var widgets []diagram.WidgetType
	parts := strings.Split(widgetsStr, ",")
	for _, part := range parts {
		name := strings.TrimSpace(strings.ToLower(part))
		if widget, ok := widgetMap[name]; ok {
			widgets = append(widgets, widget)
		}
	}

	if len(widgets) == 0 {
		return diagram.DefaultConfig().Widgets
	}

	return widgets
}

func newToolResultError(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: message,
			},
		},
		IsError: true,
	}
}

func buildSummary(arch *analyzer.Architecture, outputPath string, config diagram.HTMLConfig) string {
	counts := make(map[analyzer.ComponentType]int)
	for _, comp := range arch.Components {
		counts[comp.Type]++
	}

	summary := fmt.Sprintf("Interactive architecture report generated!\n\nOutput: %s\nTheme: %s\n\nComponents found:\n", outputPath, config.Theme)

	// List components in layer order
	typeLabels := []struct {
		Type  analyzer.ComponentType
		Label string
	}{
		{analyzer.ComponentHandler, "Handlers (Transport)"},
		{analyzer.ComponentService, "Services (Business Logic)"},
		{analyzer.ComponentAdapter, "Adapters (External)"},
		{analyzer.ComponentRepository, "Repositories (Data)"},
	}

	for _, tl := range typeLabels {
		if count, ok := counts[tl.Type]; ok && count > 0 {
			summary += fmt.Sprintf("  - %s: %d\n", tl.Label, count)
		}
	}

	// List dependency connections
	depCount := 0
	for _, deps := range arch.Dependencies {
		depCount += len(deps)
	}
	if depCount > 0 {
		summary += fmt.Sprintf("\nDependencies: %d connections\n", depCount)
	}

	// List included widgets
	summary += "\nIncluded visualizations:\n"
	for _, w := range config.Widgets {
		summary += fmt.Sprintf("  - %s\n", w)
	}

	return summary
}
