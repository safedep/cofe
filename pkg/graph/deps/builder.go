package builder

import (
	"fmt"
	"path"

	"github.com/google/osv-scanner/pkg/lockfile"
	"github.com/hmdsefi/gograph"
	"github.com/safedep/deps_weaver/pkg/core/models"
	"github.com/safedep/deps_weaver/pkg/vet"
	"github.com/safedep/dry/log"
)

type NodeTypeEnum string

const (
	PackageNodeType = NodeTypeEnum("PackageNode")
)

// Graph Nodes
type IDepNode interface {
	Key() string
	GetNodeType() NodeTypeEnum
}

// Graph Pkg Node
type PkgNode struct {
	pkg *models.Package
}

// Get the unique key of the node, used for loopups
func (d *PkgNode) Key() string {
	return fmt.Sprintf("%s:%s:%s", d.pkg.PackageDetails.Name, d.pkg.PackageDetails.Version, d.pkg.PackageDetails.Ecosystem)
}

// Get Node type of the node
func (d *PkgNode) GetNodeType() NodeTypeEnum {
	return PackageNodeType
}

type DepsCrawler struct {
	vetScanner  *vet.VetScanner
	rootPkgNode *PkgNode
}

type recursiveCrawler struct {
	graph        gograph.Graph[IDepNode]
	visitedNodes map[string]*gograph.Vertex[IDepNode]
	queue        []*gograph.Vertex[IDepNode]
	index        int
}

func NewRecursiveCrawler(graph gograph.Graph[IDepNode]) *recursiveCrawler {
	return &recursiveCrawler{
		graph:        graph,
		visitedNodes: make(map[string]*gograph.Vertex[IDepNode], 0),
	}
}

func (r *recursiveCrawler) isVisited(n IDepNode) bool {
	_, ok := r.visitedNodes[n.Key()]
	return ok
}

func (r *recursiveCrawler) markVisited(n *gograph.Vertex[IDepNode]) {
	r.visitedNodes[n.Label().Key()] = n
}

func (r *recursiveCrawler) addNode2Queue(n *gograph.Vertex[IDepNode]) {
	r.queue = append(r.queue, n)
}

func NewDepsCrawler(vi *vet.VetInput) *DepsCrawler {
	vetScanner := vet.NewVetScanner(vi)
	rootNode := createRootPackage(vi)
	crawler := DepsCrawler{vetScanner: vetScanner, rootPkgNode: rootNode}
	return &crawler
}

func createRootPackage(vi *vet.VetInput) *PkgNode {
	mani := models.Manifest{Path: vi.BaseDirectory,
		DisplayPath: vi.BaseDirectory,
		Ecosystem:   string(lockfile.PipEcosystem)}
	_, name := path.Split(vi.BaseDirectory)
	pd := models.PackageDetails{Name: name}
	return &PkgNode{pkg: &models.Package{PackageDetails: pd, Manifest: &mani}}
}

func (c *DepsCrawler) Crawl() error {
	graph := gograph.New[IDepNode](gograph.Directed())
	recCrawler := NewRecursiveCrawler(graph)
	rootVertex := graph.AddVertexByLabel(c.rootPkgNode)
	recCrawler.addNode2Queue(rootVertex)

	// Scan the base Project to find dependencies
	vetReport, err := c.vetScanner.StartScan()
	if err != nil {
		log.Debugf("Error while extracting dependencies from base project via vet %s", err)
		return err
	}

	pkgs := vetReport.GetPackages()
	for _, pkg := range pkgs.GetPackages() {
		n := &PkgNode{pkg: pkg}
		v := graph.AddVertexByLabel(n)
		graph.AddEdge(rootVertex, v)
		recCrawler.addNode2Queue(v)
	}

	//Do the recursive crawling based on the seed nodes
	recCrawler.crawl()
	return nil
}

func (r *recursiveCrawler) crawl() error {
	for r.index < len(r.queue) {
		vertex := r.queue[r.index]
		if !r.isVisited(vertex.Label()) {
			log.Debugf("Picked up node %s for processing..", vertex.Label().Key())
			r.markVisited(vertex)
			nodes := r.processNode(vertex.Label())
			for _, n := range nodes {
				log.Debugf("\tAdding node %s to the queue..", n.Key())
				if !r.isVisited(n) {
					nextVertex := r.graph.AddVertexByLabel(n)
					r.graph.AddEdge(vertex, nextVertex)
					r.addNode2Queue(nextVertex)
				}
			}
		}
		r.index += 1
	}
	return nil
}

func (r *recursiveCrawler) processNode(node IDepNode) []IDepNode {
	pkg, ok := node.(*PkgNode)
	if ok {
		pd := pkg.pkg.PackageDetails
		fmt.Println(pd)
	} else {
		log.Debugf("Found Non Pkg Node not handeled while crawling")
	}

	return make([]IDepNode, 0)
}
