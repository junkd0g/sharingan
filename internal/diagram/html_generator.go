package diagram

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/junkd0g/sharingan/internal/analyzer"
)

// WidgetType defines available visualization widgets.
type WidgetType string

const (
	WidgetArchitectureGraph WidgetType = "architecture_graph"
	WidgetComponentsPie     WidgetType = "components_pie"
	WidgetDependenciesBar   WidgetType = "dependencies_bar"
	WidgetComponentsTable   WidgetType = "components_table"
	WidgetLayerFlow         WidgetType = "layer_flow"
	WidgetStatsCards        WidgetType = "stats_cards"
	WidgetDependencyMatrix  WidgetType = "dependency_matrix"
	WidgetPackageTree       WidgetType = "package_tree"
)

// HTMLConfig configures what to include in the HTML report.
type HTMLConfig struct {
	Title       string
	Description string
	Widgets     []WidgetType
	Theme       string // "dark" or "light"
}

// DefaultConfig returns a full-featured default configuration.
func DefaultConfig() HTMLConfig {
	return HTMLConfig{
		Title:       "Go Architecture Report",
		Description: "Interactive architecture visualization",
		Theme:       "dark",
		Widgets: []WidgetType{
			WidgetStatsCards,
			WidgetArchitectureGraph,
			WidgetComponentsPie,
			WidgetDependenciesBar,
			WidgetLayerFlow,
			WidgetDependencyMatrix,
			WidgetComponentsTable,
		},
	}
}

// HTMLBuilder builds HTML reports dynamically.
type HTMLBuilder struct {
	arch   *analyzer.Architecture
	config HTMLConfig
	data   *ReportData
}

// ReportData holds all computed data for the report.
type ReportData struct {
	Components []ComponentData `json:"components"`
	Graph      GraphData       `json:"graph"`
	Stats      StatsData       `json:"stats"`
	Layers     []LayerData     `json:"layers"`
	Matrix     MatrixData      `json:"matrix"`
	Packages   []PackageData   `json:"packages"`
}

type ComponentData struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Package      string   `json:"package"`
	FilePath     string   `json:"filePath"`
	Dependencies []string `json:"dependencies"`
	DependedBy   []string `json:"dependedBy"`
	Color        string   `json:"color"`
	Category     int      `json:"category"`
}

type GraphData struct {
	Nodes      []GraphNode     `json:"nodes"`
	Links      []GraphLink     `json:"links"`
	Categories []GraphCategory `json:"categories"`
}

type GraphNode struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category int    `json:"category"`
	Value    int    `json:"value"`
	Package  string `json:"package"`
}

type GraphLink struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type GraphCategory struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type StatsData struct {
	TotalComponents  int            `json:"totalComponents"`
	TotalDeps        int            `json:"totalDependencies"`
	ComponentsByType map[string]int `json:"componentsByType"`
	AvgDependencies  float64        `json:"avgDependencies"`
	MaxDependencies  int            `json:"maxDependencies"`
	MostConnected    string         `json:"mostConnected"`
	PackageCount     int            `json:"packageCount"`
}

type LayerData struct {
	Name       string   `json:"name"`
	Color      string   `json:"color"`
	Components []string `json:"components"`
	Order      int      `json:"order"`
}

type MatrixData struct {
	Labels []string `json:"labels"`
	Data   [][]int  `json:"data"`
}

type PackageData struct {
	Name       string   `json:"name"`
	Components []string `json:"components"`
	Children   []string `json:"children"`
}

var categoryMap = map[analyzer.ComponentType]int{
	analyzer.ComponentHandler:    0,
	analyzer.ComponentService:    1,
	analyzer.ComponentRepository: 2,
	analyzer.ComponentAdapter:    3,
}

var colorMap = map[analyzer.ComponentType]string{
	analyzer.ComponentHandler:    "#4A90D9",
	analyzer.ComponentService:    "#50C878",
	analyzer.ComponentRepository: "#FFB347",
	analyzer.ComponentAdapter:    "#9B59B6",
}

var typeLabels = map[analyzer.ComponentType]string{
	analyzer.ComponentHandler:    "Handler",
	analyzer.ComponentService:    "Service",
	analyzer.ComponentRepository: "Repository",
	analyzer.ComponentAdapter:    "Adapter",
}

var layerOrder = map[analyzer.ComponentType]int{
	analyzer.ComponentHandler:    0,
	analyzer.ComponentService:    1,
	analyzer.ComponentAdapter:    2,
	analyzer.ComponentRepository: 3,
}

// GenerateHTML creates an interactive HTML report from the architecture.
func GenerateHTML(arch *analyzer.Architecture, outputPath string, config HTMLConfig) error {
	builder := &HTMLBuilder{
		arch:   arch,
		config: config,
	}

	// Build all data
	builder.data = builder.buildReportData()

	// Generate HTML
	html := builder.render()

	if err := writeFileBytes(outputPath, []byte(html)); err != nil {
		return fmt.Errorf("failed to write HTML file: %w", err)
	}

	return nil
}

func (b *HTMLBuilder) buildReportData() *ReportData {
	data := &ReportData{
		Components: b.buildComponentData(),
		Stats:      b.buildStatsData(),
		Layers:     b.buildLayerData(),
	}
	data.Graph = b.buildGraphData(data.Components)
	data.Matrix = b.buildMatrixData(data.Components)
	data.Packages = b.buildPackageData()
	return data
}

func (b *HTMLBuilder) buildComponentData() []ComponentData {
	// Build reverse dependency map
	dependedBy := make(map[string][]string)
	for _, comp := range b.arch.Components {
		for _, dep := range comp.Dependencies {
			dependedBy[dep] = append(dependedBy[dep], comp.Name)
		}
	}

	components := make([]ComponentData, 0, len(b.arch.Components))
	for _, comp := range b.arch.Components {
		components = append(components, ComponentData{
			Name:         comp.Name,
			Type:         string(comp.Type),
			Package:      comp.Package,
			FilePath:     comp.FilePath,
			Dependencies: comp.Dependencies,
			DependedBy:   dependedBy[comp.Name],
			Color:        colorMap[comp.Type],
			Category:     categoryMap[comp.Type],
		})
	}
	return components
}

func (b *HTMLBuilder) buildStatsData() StatsData {
	stats := StatsData{
		TotalComponents:  len(b.arch.Components),
		ComponentsByType: make(map[string]int),
	}

	packages := make(map[string]bool)
	totalDeps := 0
	maxDeps := 0
	mostConnected := ""

	for _, comp := range b.arch.Components {
		label := typeLabels[comp.Type]
		stats.ComponentsByType[label]++
		packages[comp.Package] = true

		deps := len(comp.Dependencies)
		totalDeps += deps
		if deps > maxDeps {
			maxDeps = deps
			mostConnected = comp.Name
		}
	}

	stats.TotalDeps = totalDeps
	stats.MaxDependencies = maxDeps
	stats.MostConnected = mostConnected
	stats.PackageCount = len(packages)

	if len(b.arch.Components) > 0 {
		stats.AvgDependencies = float64(totalDeps) / float64(len(b.arch.Components))
	}

	return stats
}

func (b *HTMLBuilder) buildLayerData() []LayerData {
	layerMap := make(map[analyzer.ComponentType][]string)
	for _, comp := range b.arch.Components {
		layerMap[comp.Type] = append(layerMap[comp.Type], comp.Name)
	}

	layers := []LayerData{}
	types := []analyzer.ComponentType{
		analyzer.ComponentHandler,
		analyzer.ComponentService,
		analyzer.ComponentAdapter,
		analyzer.ComponentRepository,
	}

	for _, t := range types {
		if comps, ok := layerMap[t]; ok && len(comps) > 0 {
			layers = append(layers, LayerData{
				Name:       typeLabels[t],
				Color:      colorMap[t],
				Components: comps,
				Order:      layerOrder[t],
			})
		}
	}

	return layers
}

func (b *HTMLBuilder) buildGraphData(components []ComponentData) GraphData {
	data := GraphData{
		Nodes: make([]GraphNode, 0, len(components)),
		Links: make([]GraphLink, 0),
		Categories: []GraphCategory{
			{Name: "Handler", Color: "#4A90D9"},
			{Name: "Service", Color: "#50C878"},
			{Name: "Repository", Color: "#FFB347"},
			{Name: "Adapter", Color: "#9B59B6"},
		},
	}

	for _, comp := range components {
		data.Nodes = append(data.Nodes, GraphNode{
			ID:       comp.Name,
			Name:     comp.Name,
			Category: comp.Category,
			Value:    len(comp.Dependencies) + len(comp.DependedBy) + 1,
			Package:  comp.Package,
		})

		for _, dep := range comp.Dependencies {
			data.Links = append(data.Links, GraphLink{
				Source: comp.Name,
				Target: dep,
			})
		}
	}

	return data
}

func (b *HTMLBuilder) buildMatrixData(components []ComponentData) MatrixData {
	n := len(components)
	labels := make([]string, n)
	nameToIdx := make(map[string]int)

	for i, comp := range components {
		labels[i] = comp.Name
		nameToIdx[comp.Name] = i
	}

	// Initialize matrix
	matrix := make([][]int, n)
	for i := range matrix {
		matrix[i] = make([]int, n)
	}

	// Fill matrix
	for i, comp := range components {
		for _, dep := range comp.Dependencies {
			if j, ok := nameToIdx[dep]; ok {
				matrix[i][j] = 1
			}
		}
	}

	return MatrixData{
		Labels: labels,
		Data:   matrix,
	}
}

func (b *HTMLBuilder) buildPackageData() []PackageData {
	pkgMap := make(map[string][]string)
	for _, comp := range b.arch.Components {
		pkgMap[comp.Package] = append(pkgMap[comp.Package], comp.Name)
	}

	packages := make([]PackageData, 0, len(pkgMap))
	for name, comps := range pkgMap {
		packages = append(packages, PackageData{
			Name:       name,
			Components: comps,
		})
	}
	return packages
}

func (b *HTMLBuilder) render() string {
	var sb strings.Builder

	// Write HTML head
	sb.WriteString(b.renderHead())

	// Write body open and container
	sb.WriteString(`<body><div class="container">`)

	// Header
	sb.WriteString(b.renderHeader())

	// Render requested widgets
	for _, widget := range b.config.Widgets {
		sb.WriteString(b.renderWidget(widget))
	}

	// Footer
	sb.WriteString(b.renderFooter())

	// Close container and body
	sb.WriteString(`</div>`)

	// Write scripts
	sb.WriteString(b.renderScripts())

	sb.WriteString(`</body></html>`)

	return sb.String()
}

func (b *HTMLBuilder) renderHead() string {
	theme := b.getThemeCSS()
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s</title>
    <script src="https://cdn.jsdelivr.net/npm/echarts@5.4.3/dist/echarts.min.js"></script>
    <style>%s</style>
</head>`, b.config.Title, theme)
}

func (b *HTMLBuilder) getThemeCSS() string {
	if b.config.Theme == "light" {
		return lightThemeCSS
	}
	return darkThemeCSS
}

func (b *HTMLBuilder) renderHeader() string {
	return fmt.Sprintf(`
<header>
    <h1>%s</h1>
    <p>%s</p>
</header>`, b.config.Title, b.config.Description)
}

func (b *HTMLBuilder) renderFooter() string {
	return `<footer><p>Generated by Sharingan - Go Architecture Analyzer</p></footer>`
}

func (b *HTMLBuilder) renderWidget(widget WidgetType) string {
	switch widget {
	case WidgetStatsCards:
		return b.renderStatsCards()
	case WidgetArchitectureGraph:
		return b.renderArchitectureGraph()
	case WidgetComponentsPie:
		return b.renderComponentsPie()
	case WidgetDependenciesBar:
		return b.renderDependenciesBar()
	case WidgetLayerFlow:
		return b.renderLayerFlow()
	case WidgetDependencyMatrix:
		return b.renderDependencyMatrix()
	case WidgetComponentsTable:
		return b.renderComponentsTable()
	case WidgetPackageTree:
		return b.renderPackageTree()
	default:
		return ""
	}
}

func (b *HTMLBuilder) renderStatsCards() string {
	return fmt.Sprintf(`
<div class="widget stats-grid">
    <div class="stat-card">
        <div class="number">%d</div>
        <div class="label">Components</div>
    </div>
    <div class="stat-card">
        <div class="number">%d</div>
        <div class="label">Dependencies</div>
    </div>
    <div class="stat-card">
        <div class="number">%d</div>
        <div class="label">Packages</div>
    </div>
    <div class="stat-card">
        <div class="number">%.1f</div>
        <div class="label">Avg Deps</div>
    </div>
</div>`,
		b.data.Stats.TotalComponents,
		b.data.Stats.TotalDeps,
		b.data.Stats.PackageCount,
		b.data.Stats.AvgDependencies)
}

func (b *HTMLBuilder) renderArchitectureGraph() string {
	return `
<div class="widget chart-box">
    <h3>Architecture Graph</h3>
    <div id="architecture-graph" class="chart-large"></div>
    <div class="legend">
        <div class="legend-item"><div class="legend-color" style="background:#4A90D9"></div><span>Handler</span></div>
        <div class="legend-item"><div class="legend-color" style="background:#50C878"></div><span>Service</span></div>
        <div class="legend-item"><div class="legend-color" style="background:#FFB347"></div><span>Repository</span></div>
        <div class="legend-item"><div class="legend-color" style="background:#9B59B6"></div><span>Adapter</span></div>
    </div>
</div>`
}

func (b *HTMLBuilder) renderComponentsPie() string {
	return `
<div class="widget chart-box half">
    <h3>Components by Type</h3>
    <div id="components-pie" class="chart"></div>
</div>`
}

func (b *HTMLBuilder) renderDependenciesBar() string {
	return `
<div class="widget chart-box half">
    <h3>Top Dependencies</h3>
    <div id="dependencies-bar" class="chart"></div>
</div>`
}

func (b *HTMLBuilder) renderLayerFlow() string {
	return `
<div class="widget chart-box">
    <h3>Layer Flow</h3>
    <div id="layer-flow" class="chart-large"></div>
</div>`
}

func (b *HTMLBuilder) renderDependencyMatrix() string {
	if len(b.data.Components) > 20 {
		return "" // Skip for large architectures
	}
	return `
<div class="widget chart-box">
    <h3>Dependency Matrix</h3>
    <div id="dependency-matrix" class="chart-large"></div>
</div>`
}

func (b *HTMLBuilder) renderComponentsTable() string {
	var rows strings.Builder
	for _, comp := range b.data.Components {
		deps := strings.Join(comp.Dependencies, ", ")
		if deps == "" {
			deps = "-"
		}
		rows.WriteString(fmt.Sprintf(`
        <tr>
            <td><strong>%s</strong></td>
            <td><span class="badge" style="background:%s22;color:%s">%s</span></td>
            <td>%s</td>
            <td>%d</td>
            <td class="deps-cell">%s</td>
        </tr>`,
			comp.Name, comp.Color, comp.Color, comp.Type,
			comp.Package, len(comp.Dependencies), deps))
	}

	return fmt.Sprintf(`
<div class="widget table-box">
    <h3>All Components</h3>
    <table>
        <thead>
            <tr><th>Name</th><th>Type</th><th>Package</th><th>Deps</th><th>Dependencies</th></tr>
        </thead>
        <tbody>%s</tbody>
    </table>
</div>`, rows.String())
}

func (b *HTMLBuilder) renderPackageTree() string {
	return `
<div class="widget chart-box">
    <h3>Package Structure</h3>
    <div id="package-tree" class="chart-large"></div>
</div>`
}

func (b *HTMLBuilder) renderScripts() string {
	dataJSON, _ := json.Marshal(b.data)

	// Determine which charts to initialize based on widgets
	var chartInits strings.Builder

	for _, widget := range b.config.Widgets {
		switch widget {
		case WidgetArchitectureGraph:
			chartInits.WriteString(architectureGraphScript)
		case WidgetComponentsPie:
			chartInits.WriteString(componentsPieScript)
		case WidgetDependenciesBar:
			chartInits.WriteString(dependenciesBarScript)
		case WidgetLayerFlow:
			chartInits.WriteString(layerFlowScript)
		case WidgetDependencyMatrix:
			if len(b.data.Components) <= 20 {
				chartInits.WriteString(dependencyMatrixScript)
			}
		case WidgetPackageTree:
			chartInits.WriteString(packageTreeScript)
		}
	}

	return fmt.Sprintf(`
<script>
const data = %s;
const charts = [];

%s

window.addEventListener('resize', () => charts.forEach(c => c.resize()));
</script>`, string(dataJSON), chartInits.String())
}

// Chart initialization scripts
const architectureGraphScript = `
(function() {
    const el = document.getElementById('architecture-graph');
    if (!el) return;
    const chart = echarts.init(el);
    charts.push(chart);
    chart.setOption({
        tooltip: {
            trigger: 'item',
            formatter: p => p.dataType === 'node'
                ? '<strong>' + p.data.name + '</strong><br/>Package: ' + p.data.package
                : p.data.source + ' → ' + p.data.target
        },
        series: [{
            type: 'graph',
            layout: 'force',
            roam: true,
            draggable: true,
            data: data.graph.nodes.map(n => ({
                ...n,
                symbolSize: Math.max(35, n.value * 12),
                itemStyle: { color: data.graph.categories[n.category].color },
                label: { show: true, position: 'bottom', formatter: n.name, fontSize: 11, color: '#aaa' }
            })),
            links: data.graph.links.map(l => ({
                ...l,
                lineStyle: { color: '#555', width: 2, curveness: 0.2 }
            })),
            categories: data.graph.categories,
            force: { repulsion: 400, gravity: 0.1, edgeLength: [80, 180] },
            emphasis: { focus: 'adjacency', lineStyle: { width: 4 } }
        }]
    });
})();
`

const componentsPieScript = `
(function() {
    const el = document.getElementById('components-pie');
    if (!el) return;
    const chart = echarts.init(el);
    charts.push(chart);
    const colors = { Handler: '#4A90D9', Service: '#50C878', Repository: '#FFB347', Adapter: '#9B59B6' };
    chart.setOption({
        tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
        series: [{
            type: 'pie',
            radius: ['40%', '70%'],
            itemStyle: { borderRadius: 8, borderColor: '#1a1a2e', borderWidth: 2 },
            label: { color: '#aaa' },
            data: Object.entries(data.stats.componentsByType).map(([name, value]) => ({
                name, value, itemStyle: { color: colors[name] }
            }))
        }]
    });
})();
`

const dependenciesBarScript = `
(function() {
    const el = document.getElementById('dependencies-bar');
    if (!el) return;
    const chart = echarts.init(el);
    charts.push(chart);
    const sorted = [...data.components].sort((a, b) => b.dependencies.length - a.dependencies.length).slice(0, 10);
    chart.setOption({
        tooltip: { trigger: 'axis' },
        grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
        xAxis: { type: 'value', axisLine: { lineStyle: { color: '#555' } }, axisLabel: { color: '#888' }, splitLine: { lineStyle: { color: '#333' } } },
        yAxis: { type: 'category', data: sorted.map(c => c.name), axisLine: { lineStyle: { color: '#555' } }, axisLabel: { color: '#888' } },
        series: [{ type: 'bar', data: sorted.map(c => ({ value: c.dependencies.length, itemStyle: { color: c.color } })), barWidth: '60%', itemStyle: { borderRadius: [0, 4, 4, 0] } }]
    });
})();
`

const layerFlowScript = `
(function() {
    const el = document.getElementById('layer-flow');
    if (!el) return;
    const chart = echarts.init(el);
    charts.push(chart);

    const nodes = [];
    const links = [];

    data.layers.forEach(layer => {
        layer.components.forEach(comp => {
            nodes.push({ name: comp });
        });
    });

    data.components.forEach(comp => {
        comp.dependencies.forEach(dep => {
            links.push({ source: comp.name, target: dep, value: 1 });
        });
    });

    chart.setOption({
        tooltip: { trigger: 'item' },
        series: [{
            type: 'sankey',
            layout: 'none',
            emphasis: { focus: 'adjacency' },
            data: nodes,
            links: links,
            lineStyle: { color: 'gradient', curveness: 0.5 },
            itemStyle: { borderWidth: 1, borderColor: '#aaa' },
            label: { color: '#ccc' }
        }]
    });
})();
`

const dependencyMatrixScript = `
(function() {
    const el = document.getElementById('dependency-matrix');
    if (!el) return;
    const chart = echarts.init(el);
    charts.push(chart);

    const matrixData = [];
    data.matrix.data.forEach((row, i) => {
        row.forEach((val, j) => {
            matrixData.push([j, i, val]);
        });
    });

    chart.setOption({
        tooltip: {
            formatter: p => p.data[2] ? data.matrix.labels[p.data[1]] + ' → ' + data.matrix.labels[p.data[0]] : ''
        },
        grid: { top: '10%', left: '15%', right: '5%', bottom: '15%' },
        xAxis: { type: 'category', data: data.matrix.labels, axisLabel: { rotate: 45, color: '#888', fontSize: 10 }, axisLine: { lineStyle: { color: '#555' } } },
        yAxis: { type: 'category', data: data.matrix.labels, axisLabel: { color: '#888', fontSize: 10 }, axisLine: { lineStyle: { color: '#555' } } },
        visualMap: { show: false, min: 0, max: 1, inRange: { color: ['#1a1a2e', '#50C878'] } },
        series: [{ type: 'heatmap', data: matrixData, itemStyle: { borderColor: '#333', borderWidth: 1 } }]
    });
})();
`

const packageTreeScript = `
(function() {
    const el = document.getElementById('package-tree');
    if (!el) return;
    const chart = echarts.init(el);
    charts.push(chart);

    const treeData = {
        name: 'packages',
        children: data.packages.map(pkg => ({
            name: pkg.name,
            children: pkg.components.map(c => ({ name: c }))
        }))
    };

    chart.setOption({
        tooltip: { trigger: 'item' },
        series: [{
            type: 'tree',
            data: [treeData],
            top: '10%', left: '10%', bottom: '10%', right: '10%',
            symbol: 'circle',
            symbolSize: 10,
            orient: 'TB',
            label: { position: 'bottom', rotate: 0, fontSize: 11, color: '#aaa' },
            leaves: { label: { position: 'bottom' } },
            expandAndCollapse: true,
            animationDuration: 500,
            lineStyle: { color: '#555', width: 1.5, curveness: 0.5 }
        }]
    });
})();
`

// Theme CSS
const darkThemeCSS = `
* { margin: 0; padding: 0; box-sizing: border-box; }
body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
    min-height: 100vh;
    color: #e4e4e4;
}
.container { max-width: 1600px; margin: 0 auto; padding: 20px; }
header { text-align: center; padding: 30px 0; border-bottom: 1px solid #333; margin-bottom: 30px; }
header h1 { font-size: 2.5rem; background: linear-gradient(90deg, #4A90D9, #50C878); -webkit-background-clip: text; -webkit-text-fill-color: transparent; margin-bottom: 10px; }
header p { color: #888; font-size: 1.1rem; }
.widget { margin-bottom: 25px; }
.stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 20px; }
.stat-card { background: rgba(255,255,255,0.05); border-radius: 12px; padding: 20px; text-align: center; border: 1px solid rgba(255,255,255,0.1); transition: transform 0.2s; }
.stat-card:hover { transform: translateY(-5px); }
.stat-card .number { font-size: 2.5rem; font-weight: bold; background: linear-gradient(90deg, #4A90D9, #50C878); -webkit-background-clip: text; -webkit-text-fill-color: transparent; }
.stat-card .label { color: #888; margin-top: 5px; }
.chart-box { background: rgba(255,255,255,0.05); border-radius: 12px; padding: 20px; border: 1px solid rgba(255,255,255,0.1); }
.chart-box.half { display: inline-block; width: calc(50% - 12px); vertical-align: top; }
.chart-box.half:nth-of-type(odd) { margin-right: 20px; }
@media (max-width: 900px) { .chart-box.half { width: 100%; margin-right: 0; } }
.chart-box h3 { margin-bottom: 15px; color: #fff; font-size: 1.2rem; }
.chart { width: 100%; height: 350px; }
.chart-large { width: 100%; height: 500px; }
.legend { display: flex; justify-content: center; gap: 25px; margin-top: 15px; flex-wrap: wrap; }
.legend-item { display: flex; align-items: center; gap: 8px; }
.legend-color { width: 14px; height: 14px; border-radius: 3px; }
.table-box { background: rgba(255,255,255,0.05); border-radius: 12px; padding: 20px; border: 1px solid rgba(255,255,255,0.1); overflow-x: auto; }
.table-box h3 { margin-bottom: 15px; color: #fff; font-size: 1.2rem; }
table { width: 100%; border-collapse: collapse; }
th, td { padding: 12px 15px; text-align: left; border-bottom: 1px solid rgba(255,255,255,0.1); }
th { background: rgba(255,255,255,0.05); font-weight: 600; }
tr:hover { background: rgba(255,255,255,0.03); }
.badge { display: inline-block; padding: 4px 12px; border-radius: 20px; font-size: 0.85rem; font-weight: 500; }
.deps-cell { font-size: 0.85rem; color: #888; max-width: 300px; }
footer { text-align: center; padding: 30px 0; color: #666; border-top: 1px solid #333; margin-top: 30px; }
`

const lightThemeCSS = `
* { margin: 0; padding: 0; box-sizing: border-box; }
body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    background: linear-gradient(135deg, #f5f7fa 0%, #e4e8ec 100%);
    min-height: 100vh;
    color: #333;
}
.container { max-width: 1600px; margin: 0 auto; padding: 20px; }
header { text-align: center; padding: 30px 0; border-bottom: 1px solid #ddd; margin-bottom: 30px; }
header h1 { font-size: 2.5rem; background: linear-gradient(90deg, #4A90D9, #50C878); -webkit-background-clip: text; -webkit-text-fill-color: transparent; margin-bottom: 10px; }
header p { color: #666; font-size: 1.1rem; }
.widget { margin-bottom: 25px; }
.stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 20px; }
.stat-card { background: #fff; border-radius: 12px; padding: 20px; text-align: center; border: 1px solid #e0e0e0; box-shadow: 0 2px 8px rgba(0,0,0,0.05); transition: transform 0.2s; }
.stat-card:hover { transform: translateY(-5px); box-shadow: 0 8px 20px rgba(0,0,0,0.1); }
.stat-card .number { font-size: 2.5rem; font-weight: bold; background: linear-gradient(90deg, #4A90D9, #50C878); -webkit-background-clip: text; -webkit-text-fill-color: transparent; }
.stat-card .label { color: #666; margin-top: 5px; }
.chart-box { background: #fff; border-radius: 12px; padding: 20px; border: 1px solid #e0e0e0; box-shadow: 0 2px 8px rgba(0,0,0,0.05); }
.chart-box.half { display: inline-block; width: calc(50% - 12px); vertical-align: top; }
.chart-box.half:nth-of-type(odd) { margin-right: 20px; }
@media (max-width: 900px) { .chart-box.half { width: 100%; margin-right: 0; } }
.chart-box h3 { margin-bottom: 15px; color: #333; font-size: 1.2rem; }
.chart { width: 100%; height: 350px; }
.chart-large { width: 100%; height: 500px; }
.legend { display: flex; justify-content: center; gap: 25px; margin-top: 15px; flex-wrap: wrap; }
.legend-item { display: flex; align-items: center; gap: 8px; }
.legend-color { width: 14px; height: 14px; border-radius: 3px; }
.table-box { background: #fff; border-radius: 12px; padding: 20px; border: 1px solid #e0e0e0; box-shadow: 0 2px 8px rgba(0,0,0,0.05); overflow-x: auto; }
.table-box h3 { margin-bottom: 15px; color: #333; font-size: 1.2rem; }
table { width: 100%; border-collapse: collapse; }
th, td { padding: 12px 15px; text-align: left; border-bottom: 1px solid #eee; }
th { background: #f9f9f9; font-weight: 600; }
tr:hover { background: #f5f5f5; }
.badge { display: inline-block; padding: 4px 12px; border-radius: 20px; font-size: 0.85rem; font-weight: 500; }
.deps-cell { font-size: 0.85rem; color: #666; max-width: 300px; }
footer { text-align: center; padding: 30px 0; color: #999; border-top: 1px solid #ddd; margin-top: 30px; }
`
