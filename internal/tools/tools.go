package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

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
		mcp.WithDescription("Generates an architecture diagram from a Go service repository. The diagram shows handlers, services, repositories and their relationships. Supports PNG and SVG output formats."),
		mcp.WithString("repo_path",
			mcp.Required(),
			mcp.Description("The absolute path to the Go service repository to analyze"),
		),
		mcp.WithString("output_path",
			mcp.Description("The output path for the diagram file. Supports .png and .svg extensions. Defaults to ./architecture.png in the repo"),
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
	outputPath := filepath.Join(repoPath, "architecture.png")
	if op, ok := request.Params.Arguments["output_path"].(string); ok && op != "" {
		outputPath = op
	}

	// Analyze the repository
	arch, err := analyzer.Analyze(repoPath)
	if err != nil {
		return newToolResultError(fmt.Sprintf("failed to analyze repository: %v", err)), nil
	}

	if len(arch.Components) == 0 {
		return newToolResultError("no architectural components found in the repository"), nil
	}

	// Generate the diagram
	if err := diagram.Generate(arch, outputPath); err != nil {
		return newToolResultError(fmt.Sprintf("failed to generate diagram: %v", err)), nil
	}

	// Build summary
	summary := buildSummary(arch, outputPath)

	return mcp.NewToolResultText(summary), nil
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

func buildSummary(arch *analyzer.Architecture, outputPath string) string {
	counts := make(map[analyzer.ComponentType]int)
	for _, comp := range arch.Components {
		counts[comp.Type]++
	}

	summary := fmt.Sprintf("Architecture diagram generated successfully!\n\nOutput: %s\n\nComponents found:\n", outputPath)

	typeLabels := map[analyzer.ComponentType]string{
		analyzer.ComponentHandler:    "Handlers",
		analyzer.ComponentService:    "Services",
		analyzer.ComponentRepository: "Repositories",
		analyzer.ComponentModel:      "Models",
		analyzer.ComponentMiddleware: "Middleware",
		analyzer.ComponentConfig:     "Config",
		analyzer.ComponentUnknown:    "Other",
	}

	for compType, label := range typeLabels {
		if count, ok := counts[compType]; ok && count > 0 {
			summary += fmt.Sprintf("  - %s: %d\n", label, count)
		}
	}

	return summary
}
