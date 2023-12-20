package builder

import (
	"fmt"
	"strings"

	G "github.com/dominikbraun/graph"
	"github.com/safedep/deps_weaver/pkg/core/models"
)

// Sortable structure to store node, weight and path from root
type nodeWithWeight struct {
	n              iDepNode
	w              int
	pathFromSource []string
}
type byNodeWeight []*nodeWithWeight

func (a byNodeWeight) Len() int           { return len(a) }
func (a byNodeWeight) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byNodeWeight) Less(i, j int) bool { return a[i].w > a[j].w }

func newNodeWithWeight(w int, n iDepNode, pathFromSource []string) *nodeWithWeight {
	return &nodeWithWeight{n: n, w: w, pathFromSource: pathFromSource}
}

type nodeTypeEnum string
type iDepNodeGraph struct {
	g G.Graph[string, iDepNode]
	// Cached vuln and low hygiene nodes
	vulnNodes            map[string]*nodeWithWeight
	lowHygieneScoreNodes map[string]*nodeWithWeight
}

func newIDepNodeGraph(g G.Graph[string, iDepNode]) *iDepNodeGraph {
	return &iDepNodeGraph{g: g,
		vulnNodes:            make(map[string]*nodeWithWeight, 0),
		lowHygieneScoreNodes: make(map[string]*nodeWithWeight, 0),
	}
}

func (g *iDepNodeGraph) addVulnNode(n iDepNode, weight int, pathFromRoot []string) {
	g.vulnNodes[n.Key()] = newNodeWithWeight(weight, n, pathFromRoot)
}

func (g *iDepNodeGraph) addLowHygieneScoreNode(n iDepNode, weight int, pathFromRoot []string) {
	g.lowHygieneScoreNodes[n.Key()] = newNodeWithWeight(weight, n, pathFromRoot)
}

func (g *iDepNodeGraph) getVulnNodes() []*nodeWithWeight {
	var vulnNodes []*nodeWithWeight
	for _, v := range g.vulnNodes {
		vulnNodes = append(vulnNodes, v)
	}
	return vulnNodes
}

func (g *iDepNodeGraph) getLowHygieneNodes() []*nodeWithWeight {
	var vulnNodes []*nodeWithWeight
	for _, v := range g.lowHygieneScoreNodes {
		vulnNodes = append(vulnNodes, v)
	}
	return vulnNodes
}

const (
	PackageNodeType = nodeTypeEnum("PackageNode")
)

// Graph Nodes
type iDepNode interface {
	Key() string
	GetNodeType() nodeTypeEnum
	GetDepth() int
}

// Graph Pkg Node
type pkgGraphNode struct {
	pkg           models.Package
	depth         int
	vulnWeight    int
	hygieneWeight int
}

type pkgGraphEdgeData struct {
	VulnWeight    int
	HygieneWeight int
}

func iDepNodeHashFunc(n iDepNode) string {
	return n.Key()
}

func NewPkgGraphEdgeData() *pkgGraphEdgeData {
	return &pkgGraphEdgeData{}
}

// Get the unique key of the node, used for loopups
func (d *pkgGraphNode) Key() string {
	// return fmt.Sprintf("%s:%s:%s", d.pkg.PackageDetails.Name, d.pkg.PackageDetails.Version, d.pkg.PackageDetails.Ecosystem)
	return fmt.Sprintf("%s", strings.ToLower(d.pkg.PackageDetails.Name))

}

// Get Node type of the node
func (d *pkgGraphNode) GetNodeType() nodeTypeEnum {
	return PackageNodeType
}

// Get Node type of the node
func (d *pkgGraphNode) GetDepth() int {
	return d.depth
}
