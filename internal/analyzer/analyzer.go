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
	ComponentModel      ComponentType = "model"
	ComponentMiddleware ComponentType = "middleware"
	ComponentConfig     ComponentType = "config"
	ComponentUnknown    ComponentType = "unknown"
)

// Component represents an architectural component in the codebase.
type Component struct {
	Name         string
	Type         ComponentType
	Package      string
	FilePath     string
	Dependencies []string
}

// Architecture represents the analyzed architecture of a service.
type Architecture struct {
	Components   []Component
	Dependencies map[string][]string // component name -> list of dependencies
}

// Analyze analyzes a Go repository and extracts its architecture.
func Analyze(repoPath string) (*Architecture, error) {
	arch := &Architecture{
		Components:   []Component{},
		Dependencies: make(map[string][]string),
	}

	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip vendor, .git, and test files
		if info.IsDir() {
			name := info.Name()
			if name == "vendor" || name == ".git" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process Go files
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		components, err := analyzeFile(path, repoPath)
		if err != nil {
			return nil // Skip files that can't be parsed
		}

		arch.Components = append(arch.Components, components...)
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Build dependency map
	for _, comp := range arch.Components {
		arch.Dependencies[comp.Name] = comp.Dependencies
	}

	return arch, nil
}

func analyzeFile(filePath, repoPath string) ([]Component, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	relPath, _ := filepath.Rel(repoPath, filePath)
	pkgPath := filepath.Dir(relPath)
	components := []Component{}

	// Determine component type based on package/file name
	compType := detectComponentType(pkgPath, filePath)

	// Find structs and their dependencies
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.TypeSpec:
			if structType, ok := x.Type.(*ast.StructType); ok {
				comp := Component{
					Name:         x.Name.Name,
					Type:         compType,
					Package:      node.Name.Name,
					FilePath:     relPath,
					Dependencies: extractDependencies(structType),
				}
				components = append(components, comp)
			}
		}
		return true
	})

	return components, nil
}

func detectComponentType(pkgPath, filePath string) ComponentType {
	lower := strings.ToLower(pkgPath + "/" + filepath.Base(filePath))

	switch {
	case containsAny(lower, "handler", "controller", "api", "http", "grpc", "rest"):
		return ComponentHandler
	case containsAny(lower, "service", "usecase", "business"):
		return ComponentService
	case containsAny(lower, "repository", "repo", "store", "storage", "database", "db", "dal"):
		return ComponentRepository
	case containsAny(lower, "model", "entity", "domain"):
		return ComponentModel
	case containsAny(lower, "middleware", "interceptor"):
		return ComponentMiddleware
	case containsAny(lower, "config", "configuration"):
		return ComponentConfig
	default:
		return ComponentUnknown
	}
}

func containsAny(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

func extractDependencies(structType *ast.StructType) []string {
	deps := []string{}
	if structType.Fields == nil {
		return deps
	}

	for _, field := range structType.Fields.List {
		typeName := extractTypeName(field.Type)
		if typeName != "" && isArchitecturalType(typeName) {
			deps = append(deps, typeName)
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

func isArchitecturalType(name string) bool {
	lower := strings.ToLower(name)
	return containsAny(lower,
		"handler", "controller",
		"service", "usecase",
		"repository", "repo", "store",
		"client", "provider",
	)
}
