# sharingan

> **⚠️ Under Construction - Not Ready for Use ⚠️**
>
> This project is currently in early development and is not in a working state. Features described below represent the intended functionality.

Sharingan is an MCP (Model Context Protocol) server that automatically analyzes Go service repositories and generates architecture diagrams. It helps developers visualize and understand the structure of Go-based microservices by identifying key architectural components and their dependencies.

## Features

- Analyzes Go repositories to extract architectural components (handlers, services, repositories, adapters)
- Builds dependency graphs by examining struct fields and their types
- Generates visual diagrams in PNG or SVG format using Graphviz
- Filters out noise like mocks, DTOs, configs, and test files to focus on real architectural patterns

## How It Works

The tool performs a two-pass analysis of your Go codebase:

1. **First pass**: Collects all interface definitions
2. **Second pass**: Identifies architectural components based on package naming conventions and dependency patterns

Components are categorized into layers:
- **Transport Layer** (handlers in `transport`, `http`, `handler`, or `api` packages)
- **Service Layer** (services with 2+ dependencies)
- **Adapters** (clients in `adapter`, `client`, `external`, or `integration` packages)
- **Data Layer** (repositories in `persistence`, `repository`, or `repo` packages)

## Usage

Sharingan exposes a single MCP tool called `generate_architecture_diagram` that takes a repository path and generates a visual diagram of the architecture.

## Requirements

- Go 1.24+
- Graphviz (for diagram rendering)
