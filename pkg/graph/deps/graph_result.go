package builder

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dominikbraun/graph"
	G "github.com/dominikbraun/graph"
	"github.com/dominikbraun/graph/draw"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/safedep/vet/pkg/common/logger"
)

var depth2color = map[int]string{
	0: "#AEEA00", //blue
	1: "#74b9ff", //blue
	2: "#7e57c2", //yellow
	3: "#d500f9", // yellow
	4: "#d500f9", // yellow
	5: "#d500f9", // yellow

}

var score2Color = map[int]string{
	0:  "#d500f9", //blue
	1:  "#d500f9", //blue
	2:  "#d500f9", //blue
	3:  "#fdd835", // Purple
	4:  "#fdd835", // Purple
	5:  "#fdd835", // Purple
	6:  "#E64A19", // Orange
	7:  "#E64A19", // Orange
	8:  "#E64A19", // Orange
	9:  "#D32F2F", // Orange
	10: "#D32F2F", // Red
}

var scorecard2Color = map[int]string{
	0:  "#d500f9", //blue
	1:  "#d500f9", //blue
	2:  "#d500f9", //blue
	3:  "#f48fb1", // pink
	4:  "#f48fb1", // pink
	5:  "#f48fb1", // pink
	6:  "#ff4081", // pink
	7:  "#ff4081", // pink
	8:  "#ff4081", // pink
	9:  "#e91e63", // pink
	10: "#e91e63", // pink Shade

}

type graphResult struct {
	rootNode iDepNode
	graph    *iDepNodeGraph
	// import2PkgMap map[string][]iDepNode
	export2PkgMap        map[string][]iDepNode
	cachedReachableGraph *iDepNodeGraph
}

func newGraphResult(rootNode iDepNode, graph *iDepNodeGraph) *graphResult {
	gres := graphResult{rootNode: rootNode, graph: graph,
		export2PkgMap: make(map[string][]iDepNode, 0),
	}

	gres.addExportedModulesMap()
	return &gres
}

func (g *graphResult) addExportedModulesMap() {
	// Post Processing of the graph
	_ = G.DFS(g.graph.g, g.rootNode.Key(), func(value string) bool {
		node, err := g.graph.g.Vertex(value)
		if err != nil {
			return false
		}
		pkgNode, _ := node.(*pkgGraphNode)
		mods := pkgNode.pkg.GetExportedModules()
		for _, m := range mods {
			g.export2PkgMap[m] = append(g.export2PkgMap[m], node)
		}
		return false
	})
}

func (g *graphResult) removeEdgesBasedOnImportedModules(gg G.Graph[string, iDepNode]) error {
	edges, err := gg.Edges()
	if err != nil {
		logger.Debugf("Error while getting edges %s", err)
		return err
	}
	for _, edge := range edges {
		sv, _ := gg.Vertex(edge.Source)
		tv, _ := gg.Vertex(edge.Target)
		targetNode, _ := tv.(*pkgGraphNode)
		sourceNode, _ := sv.(*pkgGraphNode)
		importedModules := sourceNode.pkg.GetImportedModules()
		expotedModules := targetNode.pkg.GetExportedModules()
		// Import and export module match?
		targetPkgName := targetNode.pkg.PackageDetails.Name

		ieMatch := g.stringsIntersect(importedModules, expotedModules)
		targetInImports := g.targetPkgNameInImportedModules(targetPkgName, importedModules)
		if !targetInImports && !ieMatch {
			gg.RemoveEdge(sv.Key(), tv.Key())
			logger.Debugf("Removed Edge from %s to %s - reason imported exported modules mismatch", sv.Key(), tv.Key())
			logger.Debugf("Imported Modules at source %s, Exported Modules %s", importedModules, expotedModules)

		}
	}
	return nil
}

func (g *graphResult) targetPkgNameInImportedModules(pkgName string, modules []string) bool {
	for _, mod := range modules {
		if strings.Contains(pkgName, mod) {
			return true
		}
	}
	return false
}

func (g *graphResult) stringsIntersect(src []string, des []string) bool {
	for _, s := range src {
		for _, t := range des {
			if t == s {
				return true
			}
		}
	}
	return false
}

func (g *graphResult) getIDepNodeGraph(reachable bool) (*iDepNodeGraph, error) {
	if reachable {
		if g.cachedReachableGraph == nil {
			gg, err := g.reachableGraph()
			if err != nil {
				logger.Debugf("Error while genenerating reachable graph %s", err)
				return nil, err
			}
			if err := g.addEdgeWeightBasedOnVulnScore(gg); err != nil {
				logger.Debugf("Error while adding weight to the edges %s", err)
				return nil, err
			}
			g.cachedReachableGraph = gg
		}
		return g.cachedReachableGraph, nil
	}

	return g.graph, nil
}

func (g *graphResult) addEdgeWeightBasedOnVulnScore(gr *iDepNodeGraph) error {
	vertices, err := g.getUniqueVertices(gr)
	if err != nil {
		logger.Debugf("Error while getting unique vertices %s", err)
		return err
	}
	for _, v := range *vertices {
		if v.Key() != g.rootNode.Key() {
			pkgNode := v.(*pkgGraphNode)
			if pkgNode.pkg.GetMaxVulnScore() > 0 || pkgNode.pkg.GetReverseScorecardScore() > 0 {
				path, err := graph.ShortestPath[string](gr.g, g.rootNode.Key(), v.Key())
				if err != nil {
					logger.Debugf("Error while generating shorted path %s", err)
					return err
				}
				// ignore the errors, if any
				g.updateEdgeVulnWeight(gr, v, path)
				g.updateEdgeScorecardWeight(gr, v, path)
			}
		}
	}

	return nil
}

func (g *graphResult) updateEdgeVulnWeight(gr *iDepNodeGraph, endNode iDepNode, pathFromRoot []string) error {
	pkgNode := endNode.(*pkgGraphNode)
	vulnScore := pkgNode.pkg.GetMaxVulnScore()
	if pkgNode.pkg.GetMaxVulnScore() > 0 {
		if len(pathFromRoot) > 0 {
			l := len(pathFromRoot) - 1
			if vulnScore-l > 0 {
				w := vulnScore - l + 1 // closer the vuln, higher the weight
				gr.addVulnNode(endNode, w, pathFromRoot)
				// Update the edge weights from end node to root node
				for i, s := range pathFromRoot {
					if i < len(pathFromRoot)-1 {
						t := pathFromRoot[i+1]
						e, err := gr.g.Edge(s, t)
						if err != nil {
							logger.Debugf("Error while getting edge %s", err)
							return err
						}
						if e.Properties.Data == nil {
							e.Properties.Data = NewPkgGraphEdgeData()
						}
						data := e.Properties.Data.(*pkgGraphEdgeData)
						if w > data.VulnWeight {
							data.VulnWeight = w
						}
					}
				}

			}
		}
	}

	return nil
}

func (g *graphResult) updateEdgeScorecardWeight(gr *iDepNodeGraph, endNode iDepNode, pathFromRoot []string) error {
	pkgNode := endNode.(*pkgGraphNode)
	hygieneScore := int(pkgNode.pkg.GetReverseScorecardScore())
	if pkgNode.pkg.GetReverseScorecardScore() > 0 {
		if len(pathFromRoot) > 0 {
			l := len(pathFromRoot) - 1 + 1
			if hygieneScore-l > 0 {
				w := hygieneScore - l // closer the node, higher the weight
				// Update lowHydiene Score Nodes
				gr.addLowHygieneScoreNode(endNode, w, pathFromRoot)

				// Update all the edges from end node to source
				for i, s := range pathFromRoot {
					if i < len(pathFromRoot)-1 {
						t := pathFromRoot[i+1]
						e, err := gr.g.Edge(s, t)
						if err != nil {
							logger.Debugf("Error while getting edge %s", err)
							return err
						}
						if e.Properties.Data == nil {
							e.Properties.Data = NewPkgGraphEdgeData()
						}
						data := e.Properties.Data.(*pkgGraphEdgeData)
						if w > data.HygieneWeight {
							data.HygieneWeight = w
						}
					}
				}
			}
		}
	}

	return nil
}

// useful to generate csv content
func (g *graphResult) depth2Color(depth int) string {
	c, ok := depth2color[depth]
	if !ok {
		return "#b2bec3" //grey color
	}

	return c
}

func (g *graphResult) vulnScore2Color(score int) string {
	c, ok := score2Color[score]
	if !ok {
		return "#b2bec3" //grey color
	}

	return c
}

func (g *graphResult) scorecardScore2Color(score float32) string {
	c, ok := scorecard2Color[int(score)]
	if !ok {
		return "#b2bec3" //grey color
	}

	return c
}

func (g *graphResult) node2VulnColor(n iDepNode) string {
	pkgNode, _ := n.(*pkgGraphNode)

	if n.Key() == g.rootNode.Key() {
		return "#AEEA00"
	}

	if pkgNode.pkg.GetMaxVulnScore() >= 3 {
		return g.vulnScore2Color(pkgNode.pkg.GetMaxVulnScore())
	} else {
		return g.depth2Color(pkgNode.GetDepth())
	}
}

func (g *graphResult) node2ScorecardColor(n iDepNode) string {
	pkgNode, _ := n.(*pkgGraphNode)

	if n.Key() == g.rootNode.Key() {
		return "#AEEA00"
	}

	if pkgNode.pkg.GetReverseScorecardScore() >= 3 {
		s := pkgNode.pkg.GetReverseScorecardScore()
		return g.scorecardScore2Color(s)
	} else {
		return g.depth2Color(pkgNode.GetDepth())
	}
}

func (g *graphResult) node2Color(n iDepNode) string {
	pkgNode, _ := n.(*pkgGraphNode)

	if n.Key() == g.rootNode.Key() {
		return "#AEEA00"
	}

	if pkgNode.pkg.GetMaxVulnScore() > 7 {
		return g.vulnScore2Color(pkgNode.pkg.GetMaxVulnScore())
	} else if pkgNode.pkg.GetReverseScorecardScore() > 7 {
		s := pkgNode.pkg.GetReverseScorecardScore()
		return g.scorecardScore2Color(s)
	} else {
		return g.depth2Color(pkgNode.GetDepth())
	}
}

// useful to generate csv content
func (g *graphResult) depth2timestamp(depth int) string {
	currentTime := time.Now()
	newTime := currentTime.Add(time.Duration(depth*3600) * time.Second)
	return newTime.Format(time.RFC3339)
}

func (g *graphResult) Export2Graphviz(graphviz string, reachable bool) error {
	file, _ := os.Create(graphviz)
	defer file.Close()
	gg, err := g.getIDepNodeGraph(reachable)
	if err != nil {
		return err
	}
	return draw.DOT(gg.g, file)
}

func (g *graphResult) Export2CSV(outpath string, reachable bool) error {
	gg, err := g.getIDepNodeGraph(reachable)
	if err != nil {
		return err
	}

	if err := g.exportEdges2CSV(gg, outpath); err != nil {
		logger.Debugf("Error while experting edges to csv %s", err)
		return err
	}

	metadataFile := fmt.Sprintf("%s.metadata.csv", outpath)
	if err := g.exportMetadata2CSV(gg, metadataFile); err != nil {
		logger.Debugf("Error while experting metadata to csv %s", err)
		return err
	}

	return nil

}

func (g *graphResult) exportEdges2CSV(gg *iDepNodeGraph, outpath string) error {
	edges, err := gg.g.Edges()
	if err != nil {
		logger.Debugf("Error while getting edges %s", err)
		return err
	}

	// Create or open the CSV file for writing
	file, err := os.Create(outpath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create a CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write the header row
	header := []string{"source", "target", "color", "vuln_score", "hygiene_color", "vuln_weight", "hygiene_weight", "time"}
	if err := writer.Write(header); err != nil {
		return err
	}

	i := 0
	// Write each record to the CSV file
	for _, edge := range edges {
		sv, _ := g.graph.g.Vertex(edge.Source)
		sourceNode, _ := sv.(*pkgGraphNode)
		edata := edge.Properties.Data.(*pkgGraphEdgeData)
		recordRow := []string{
			edge.Source,
			edge.Target,
			g.node2Color(sourceNode),
			g.vulnScore2Color(edata.VulnWeight),
			g.scorecardScore2Color(float32(edata.HygieneWeight)),
			strconv.Itoa(edata.VulnWeight),
			strconv.Itoa(edata.HygieneWeight),
			g.depth2timestamp(sourceNode.depth),
		}
		if err := writer.Write(recordRow); err != nil {
			return err
		}
		i += 1
	}

	return nil

}

func (g *graphResult) getUniqueVertices(gg *iDepNodeGraph) (*[]iDepNode, error) {
	edges, err := gg.g.Edges()
	if err != nil {
		logger.Debugf("Error while getting edges %s", err)
		return nil, err
	}

	var vertices []iDepNode
	cache := make(map[string]bool, 0)
	// Write each record to the CSV file
	for _, edge := range edges {
		sv, _ := g.graph.g.Vertex(edge.Source)
		_, ok := cache[edge.Source]
		if !ok {
			cache[edge.Source] = true
			vertices = append(vertices, sv)
		}

		tv, _ := g.graph.g.Vertex(edge.Target)
		_, ok = cache[edge.Target]
		if !ok {
			cache[edge.Target] = true
			vertices = append(vertices, tv)
		}
	}

	return &vertices, nil
}

func (g *graphResult) exportMetadata2CSV(gg *iDepNodeGraph, outpath string) error {
	// Create or open the CSV file for writing
	file, err := os.Create(outpath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create a CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write the header row
	header := []string{"id", "node_color", "vuln_color", "hygiene_color", "node_value", "type", "vuln_score", "hygiene_score"}
	if err := writer.Write(header); err != nil {
		return err
	}

	vertices, err := g.getUniqueVertices(gg)
	if err != nil {
		logger.Debugf("Error while getting edges %s", err)
		return err
	}

	for _, v := range *vertices {
		pkgNode := v.(*pkgGraphNode)
		recordRow := []string{
			v.Key(),
			g.node2Color(pkgNode),
			g.node2VulnColor(pkgNode),
			g.node2ScorecardColor(pkgNode),
			strconv.Itoa(v.GetDepth()),
			strconv.Itoa(v.GetDepth()),
			strconv.Itoa(pkgNode.pkg.GetMaxVulnScore()),
			strconv.Itoa(int(pkgNode.pkg.GetReverseScorecardScore())),
		}
		if err := writer.Write(recordRow); err != nil {
			logger.Warnf("Error while metadata writig to CSV file %s", err)
			return err
		}
	}
	return nil

}

func (g *graphResult) Print() {
	logger.Debugf("%s\n", g.rootNode.Key())
	_ = G.BFS(g.graph.g, g.rootNode.Key(), func(value string) bool {
		node, err := g.graph.g.Vertex(value)
		if err != nil {
			return false
		}
		node.GetDepth()
		logger.Debugf("%s %s\n", g.spaces(node.GetDepth()), node.Key())
		return false
	})
}

// getColorBasedOnScore returns the color based on the COFE priority score
func (g *graphResult) getColorBasedOnScore(score int) text.Colors {
	switch {
	case score >= 9:
		return text.Colors{text.BgRed, text.FgWhite}
	case score >= 6 && score <= 8:
		return text.Colors{text.BgHiRed, text.FgWhite}
	case score >= 3 && score < 6:
		return text.Colors{text.BgBlue, text.FgWhite}
	default:
		return text.Colors{text.BgBlack, text.FgHiWhite}
	}
}

const (
	COFE_SCORE_HEADER = "Cofe Score"
	CVSS_SCORE_HEADER = "CVSS Score"
	ECOSYSTEM_HEADER  = "Ecosystem"
	NAME_HEADER       = "Name"
	PATH_HEADER       = "Path"
)

func (g *graphResult) PrintVulns() {
	vulnNodes := g.cachedReachableGraph.getVulnNodes()
	sort.Sort(byNodeWeight(vulnNodes))

	tbl := table.NewWriter()
	tbl.SetStyle(table.StyleLight)
	tbl.SetOutputMirror(os.Stdout)

	cofePriorityTransformer := text.Transformer(func(val interface{}) string {
		w, ok := val.(int)
		if ok {
			return g.getColorBasedOnScore(w).Sprint(fmt.Sprintf("%3d%-2s", val, ""))
		} else {
			return g.getColorBasedOnScore(-1).Sprint(val)
		}
	})

	tbl.SetColumnConfigs([]table.ColumnConfig{
		{
			Name:        COFE_SCORE_HEADER,
			Align:       text.AlignCenter,
			AlignFooter: text.AlignCenter,
			AlignHeader: text.AlignCenter,
			Transformer: cofePriorityTransformer,
			WidthMin:    4,
		}, {
			Name:        CVSS_SCORE_HEADER,
			Align:       text.AlignCenter,
			AlignFooter: text.AlignCenter,
			AlignHeader: text.AlignCenter,
			Transformer: cofePriorityTransformer,
			WidthMin:    4,
		},
	})

	tbl.AppendHeader(table.Row{ECOSYSTEM_HEADER, NAME_HEADER, CVSS_SCORE_HEADER,
		COFE_SCORE_HEADER, PATH_HEADER})

	cache := make(map[string]bool, 0)
	if len(vulnNodes) > 0 {
		fmt.Println("Prioritized List of Packages to Upgrade as per Vulnerabilities: ")
		for _, v := range vulnNodes {
			cache[v.n.Key()] = true
			pkgNode, _ := (v.n).(*pkgGraphNode)
			pd := pkgNode.pkg.PackageDetails
			vulnScore := pkgNode.pkg.GetMaxVulnScore()
			path := v.pathFromSource
			tbl.AppendRow(table.Row{
				pd.Ecosystem,
				fmt.Sprintf("%s@%s", pd.Name, pd.Version),
				vulnScore,
				v.w,
				strings.Join(path, " > "),
			})
		}
	}

	origVulnNodes := g.graph.getVulnNodes()
	sort.Sort(byNodeWeight(origVulnNodes))
	if len(origVulnNodes) > 0 {
		for _, v := range origVulnNodes {
			if _, ok := cache[v.n.Key()]; !ok {
				pkgNode, _ := (v.n).(*pkgGraphNode)
				pd := pkgNode.pkg.PackageDetails
				vulnScore := pkgNode.pkg.GetMaxVulnScore()
				path := v.pathFromSource
				// Get color based on COFE Priority score
				tbl.AppendRow(table.Row{
					pd.Ecosystem,
					fmt.Sprintf("%s@%s", pd.Name, pd.Version),
					vulnScore,
					"None",
					fmt.Sprintf("None in Reduced Graph, Removed Path: %s", strings.Join(path, " > ")),
				})
			}
		}
	}

	tbl.Render()
}

func (g *graphResult) PrintLowHygieneNodes() {
	vulnNodes := g.cachedReachableGraph.getLowHygieneNodes()
	sort.Sort(byNodeWeight(vulnNodes))

	cache := make(map[string]bool, 0)
	if len(vulnNodes) > 0 {
		fmt.Println("Prioritized List of Packages to Upgrade as per Scorecard Score: ")
		for _, v := range vulnNodes {
			cache[v.n.Key()] = true
			pkgNode, _ := (v.n).(*pkgGraphNode)
			pd := pkgNode.pkg.PackageDetails
			vulnScore := pkgNode.pkg.GetScorecardScore()
			path := v.pathFromSource
			fmt.Printf("\t%s/%s [Poor Hygiene] Score [%f] Priority [%d] Path: %s\n", pd.Name, pd.Version, vulnScore, v.w, path)
		}
	}

	origVulnNodes := g.graph.getLowHygieneNodes()
	sort.Sort(byNodeWeight(origVulnNodes))
	if len(origVulnNodes) > 0 {
		fmt.Println("\nFalse Positives Removed after reachability analysis: ")
		for _, v := range origVulnNodes {
			if _, ok := cache[v.n.Key()]; !ok {
				pkgNode, _ := (v.n).(*pkgGraphNode)
				pd := pkgNode.pkg.PackageDetails
				vulnScore := pkgNode.pkg.GetScorecardScore()
				path := v.pathFromSource
				fmt.Printf("\t%s/%s [Poor Hygiene] Score [%f] Priority [%d] Path: %s\n", pd.Name, pd.Version, vulnScore, v.w, path)
			}
		}
	}
}

func (g *graphResult) reachableGraph() (*iDepNodeGraph, error) {

	gg, err := g.graph.g.Clone()
	if err != nil {
		return nil, err
	}

	// First remove edges based on the imported modules
	g.removeEdgesBasedOnImportedModules(gg)

	logger.Debugf("Reducing Reachable Graph ...")
	reachableNodes := make(map[string]bool, 0)
	reachableNodes[g.rootNode.Key()] = true
	_ = G.BFS(gg, g.rootNode.Key(), func(value string) bool {
		reachableNodes[value] = true
		return false
	})

	vmap, err := gg.AdjacencyMap()
	if err != nil {
		logger.Debugf("\n Error while getting map %s", err)
	}

	for v, edges := range vmap {
		if _, ok1 := reachableNodes[v]; !ok1 {
			for t, _ := range edges {
				// Remove both edges to and fro
				err := gg.RemoveEdge(v, t)
				if err != nil {
					logger.Debugf("Error while removing edge %s", err)
				} else {
					logger.Debugf("Removing edge %s %s", v, t)
				}
			}
		}
	}

	for v, _ := range vmap {
		if _, ok1 := reachableNodes[v]; !ok1 {
			err := gg.RemoveVertex(v)
			if err != nil {
				logger.Debugf("Error while removing vertex %s %s", v, err)
			} else {
				logger.Debugf("Removed vertex %s", v)
			}
		}
	}
	return newIDepNodeGraph(gg), nil
}

func (g *graphResult) spaces(n int) string {
	var sb strings.Builder
	for i := 0; i < n; i++ {
		sb.WriteString("..")
	}
	return sb.String()
}
