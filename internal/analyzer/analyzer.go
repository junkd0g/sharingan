package analyzer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// ComponentType represents the type of architectural component.
type ComponentType string

const (
	ComponentHandler    ComponentType = "handler"
	ComponentService    ComponentType = "service"
	ComponentRepository ComponentType = "repository"
	ComponentAdapter    ComponentType = "adapter"
)

// Component represents an architectural component in the codebase.
type Component struct {
	Name         string
	Type         ComponentType
	Package      string
	FilePath     string
	Dependencies []string // Names of dependencies (interface field types)
}

// Architecture represents the analyzed architecture of a service.
type Architecture struct {
	Components   []Component
	Dependencies map[string][]string
}

// Analyze analyzes a Go repository and extracts its core architecture.
// It focuses on finding real architectural components (handlers, services, repositories)
// and their dependencies, filtering out noise like DTOs, mocks, and configs.
func Analyze(repoPath string) (*Architecture, error) {
	arch := &Architecture{
		Components:   []Component{},
		Dependencies: make(map[string][]string),
	}

	// First pass: collect all interface names defined in the codebase
	interfaces := make(map[string]bool)
	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return skipOrContinue(info, err)
		}
		if !isGoSourceFile(path) {
			return nil
		}
		collectInterfaces(path, interfaces)
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Second pass: find architectural components
	err = filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return skipOrContinue(info, err)
		}
		if !isGoSourceFile(path) {
			return nil
		}

		components := analyzeFileForComponents(path, repoPath, interfaces)
		arch.Components = append(arch.Components, components...)
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Build dependency map and resolve dependencies to actual component names
	componentNames := make(map[string]bool)
	for _, comp := range arch.Components {
		componentNames[comp.Name] = true
	}

	// Filter dependencies to only include known components
	for i := range arch.Components {
		var validDeps []string
		for _, dep := range arch.Components[i].Dependencies {
			if componentNames[dep] {
				validDeps = append(validDeps, dep)
			}
		}
		arch.Components[i].Dependencies = validDeps
		arch.Dependencies[arch.Components[i].Name] = validDeps
	}

	return arch, nil
}

func skipOrContinue(info os.FileInfo, err error) error {
	if err != nil {
		return err
	}
	if info.IsDir() {
		name := info.Name()
		if name == "vendor" || name == ".git" || name == "node_modules" || name == "mock" || name == "mocks" {
			return filepath.SkipDir
		}
	}
	return nil
}

func isGoSourceFile(path string) bool {
	return strings.HasSuffix(path, ".go") &&
		!strings.HasSuffix(path, "_test.go") &&
		!strings.Contains(path, "/mock") &&
		!strings.Contains(path, "_mock")
}

func collectInterfaces(filePath string, interfaces map[string]bool) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return
	}

	ast.Inspect(node, func(n ast.Node) bool {
		if typeSpec, ok := n.(*ast.TypeSpec); ok {
			if _, isInterface := typeSpec.Type.(*ast.InterfaceType); isInterface {
				interfaces[typeSpec.Name.Name] = true
			}
		}
		return true
	})
}

func analyzeFileForComponents(filePath, repoPath string, interfaces map[string]bool) []Component {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil
	}

	relPath, _ := filepath.Rel(repoPath, filePath)
	pkgPath := filepath.Dir(relPath)
	var components []Component

	ast.Inspect(node, func(n ast.Node) bool {
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}

		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return true
		}

		name := typeSpec.Name.Name

		// Skip noise: mocks, DTOs, configs, internal types
		if shouldSkipStruct(name) {
			return true
		}

		// Extract interface-typed fields (these are the dependencies)
		deps := extractInterfaceDependencies(structType, interfaces)

		// Determine component type based on package path and struct characteristics
		compType := detectComponentTypeFromContext(pkgPath, name, deps)

		// Only include if it's a real architectural component
		if compType == "" {
			return true
		}

		components = append(components, Component{
			Name:         name,
			Type:         compType,
			Package:      node.Name.Name,
			FilePath:     relPath,
			Dependencies: deps,
		})

		return true
	})

	return components
}

func shouldSkipStruct(name string) bool {
	lower := strings.ToLower(name)

	// Skip mocks
	if strings.Contains(lower, "mock") {
		return true
	}

	// Skip DTOs (Request/Response structs)
	if strings.HasSuffix(name, "Request") || strings.HasSuffix(name, "Response") {
		return true
	}

	// Skip config structs (they're not architectural components)
	if strings.HasSuffix(name, "Config") || strings.HasSuffix(name, "Conf") ||
		strings.Contains(lower, "config") || name == "Config" {
		return true
	}

	// Skip common non-architectural types
	skipSuffixes := []string{"Options", "Params", "Data", "Info", "Result", "Error", "Context",
		"Structure", "Content", "Template", "Section", "Message", "Event", "Item", "Entry"}
	for _, suffix := range skipSuffixes {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}

	// Skip if name is too short (likely internal)
	if len(name) <= 2 && name != "DB" {
		return true
	}

	// Skip unexported types
	if name[0] >= 'a' && name[0] <= 'z' {
		return true
	}

	return false
}

func extractInterfaceDependencies(structType *ast.StructType, interfaces map[string]bool) []string {
	var deps []string
	if structType.Fields == nil {
		return deps
	}

	seen := make(map[string]bool)
	for _, field := range structType.Fields.List {
		typeName := extractTypeName(field.Type)
		if typeName == "" || seen[typeName] {
			continue
		}

		// Include if it's a known interface or looks like a dependency
		if interfaces[typeName] || looksLikeDependency(typeName) {
			deps = append(deps, typeName)
			seen[typeName] = true
		}
	}
	return deps
}

func extractTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return extractTypeName(t.X)
	case *ast.SelectorExpr:
		return t.Sel.Name
	default:
		return ""
	}
}

func looksLikeDependency(name string) bool {
	lower := strings.ToLower(name)
	dependencyPatterns := []string{
		"service", "store", "repo", "repository",
		"client", "api", "adapter", "provider",
		"auth", "logger", "db", "database",
		"generative", "generator",
	}
	for _, pattern := range dependencyPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

func detectComponentTypeFromContext(pkgPath, structName string, deps []string) ComponentType {
	lower := strings.ToLower(pkgPath)
	nameLower := strings.ToLower(structName)

	// Handler/Transport layer
	if strings.Contains(lower, "transport") || strings.Contains(lower, "http") ||
		strings.Contains(lower, "handler") || strings.Contains(lower, "api") ||
		strings.Contains(nameLower, "server") || strings.Contains(nameLower, "handler") {
		if len(deps) > 0 { // Handlers should have dependencies
			return ComponentHandler
		}
	}

	// Repository/Persistence layer (check before service)
	// But not if it's in a config package
	if !strings.Contains(lower, "config") {
		if strings.Contains(lower, "persistence") || strings.Contains(lower, "repository") ||
			strings.Contains(lower, "repo") || strings.Contains(lower, "store") ||
			structName == "DB" || strings.HasSuffix(structName, "Repository") ||
			strings.HasSuffix(structName, "Store") {
			return ComponentRepository
		}
	}

	// Adapter layer
	if strings.Contains(lower, "adapter") || strings.Contains(lower, "client") ||
		strings.Contains(lower, "external") || strings.Contains(lower, "integration") {
		return ComponentAdapter
	}

	// Service layer
	if strings.Contains(lower, "service") || strings.Contains(lower, "usecase") ||
		structName == "Service" || strings.HasSuffix(structName, "Service") {
		if len(deps) > 0 { // Services should have dependencies
			return ComponentService
		}
	}

	// If it has multiple dependencies, it's likely a service
	if len(deps) >= 2 {
		return ComponentService
	}

	return "" // Not an architectural component
}
