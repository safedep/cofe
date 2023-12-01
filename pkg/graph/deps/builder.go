package builder

import (
	"fmt"
	"os"
	"path"

	"github.com/google/osv-scanner/pkg/lockfile"
	"github.com/hmdsefi/gograph"
	"github.com/safedep/deps_weaver/pkg/core/models"
	"github.com/safedep/deps_weaver/pkg/pm/pypi"
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
	GetDepth() int
}

// Graph Pkg Node
type PkgNode struct {
	pkg   *models.Package
	depth int
}

// Get the unique key of the node, used for loopups
func (d *PkgNode) Key() string {
	return fmt.Sprintf("%s:%s:%s", d.pkg.PackageDetails.Name, d.pkg.PackageDetails.Version, d.pkg.PackageDetails.Ecosystem)
}

// Get Node type of the node
func (d *PkgNode) GetNodeType() NodeTypeEnum {
	return PackageNodeType
}

// Get Node type of the node
func (d *PkgNode) GetDepth() int {
	return d.depth
}

type DepsCrawler struct {
	vetScanner  *vet.VetScanner
	rootPkgNode *PkgNode
	maxDepth    int
}

type recursiveCrawler struct {
	graph        gograph.Graph[IDepNode]
	visitedNodes map[string]*gograph.Vertex[IDepNode]
	queue        []*gograph.Vertex[IDepNode]
	index        int
	pkgManager   *pypi.PypiPackageManager
	maxDepth     int
}

func (d *DepsCrawler) NewRecursiveCrawler(graph gograph.Graph[IDepNode]) *recursiveCrawler {
	pm := pypi.NewPypiPackageManager()

	return &recursiveCrawler{
		graph:        graph,
		visitedNodes: make(map[string]*gograph.Vertex[IDepNode], 0),
		pkgManager:   pm,
		maxDepth:     d.maxDepth,
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

func (r *recursiveCrawler) nextVertexFromQueue() *gograph.Vertex[IDepNode] {
	for r.index < len(r.queue) {
		vertex := r.queue[r.index]
		isVisisted := r.isVisited(vertex.Label())
		isMaxDepth := vertex.Label().GetDepth() > r.maxDepth
		if !isVisisted && !isMaxDepth {
			return vertex
		}
		r.index += 1
	}

	return nil
}

func NewDepsCrawler(vi *vet.VetInput, maxDepth int) *DepsCrawler {
	vetScanner := vet.NewVetScanner(vi)
	rootNode := createRootPackage(vi)
	crawler := DepsCrawler{vetScanner: vetScanner,
		rootPkgNode: rootNode,
		maxDepth:    maxDepth}
	return &crawler
}

func createRootPackage(vi *vet.VetInput) *PkgNode {
	mani := models.Manifest{Path: vi.BaseDirectory,
		DisplayPath: vi.BaseDirectory,
		Ecosystem:   string(lockfile.PipEcosystem)}
	_, name := path.Split(vi.BaseDirectory)
	pd := models.PackageDetails{Name: name}
	return &PkgNode{pkg: &models.Package{PackageDetails: pd, Manifest: &mani}, depth: 0}
}

func (c *DepsCrawler) Crawl() error {
	graph := gograph.New[IDepNode](gograph.Directed())
	recCrawler := c.NewRecursiveCrawler(graph)
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
		n := &PkgNode{pkg: pkg, depth: c.rootPkgNode.GetDepth() + 1}
		v := graph.AddVertexByLabel(n)
		graph.AddEdge(rootVertex, v)
		recCrawler.addNode2Queue(v)
	}

	//Do the recursive crawling based on the seed nodes
	recCrawler.crawl()
	return nil
}

func (r *recursiveCrawler) crawl() error {
	for {
		log.Debugf("Queue Length %d, Vertices %d", len(r.graph.GetAllVertices()))
		vertex := r.nextVertexFromQueue()
		if vertex == nil {
			break
		}
		fmt.Printf("Picked up node %s for processing..\n", vertex.Label().Key())
		r.markVisited(vertex)
		nodes, err := r.processNode(vertex.Label())
		if err != nil {
			log.Debugf("\tError processing node %s..", vertex.Label().Key())
		}
		for _, n := range nodes {
			fmt.Printf("\tAdding node %s to the queue..\n", n.Key())
			if !r.isVisited(n) {
				nextVertex := r.graph.AddVertexByLabel(n)
				r.graph.AddEdge(vertex, nextVertex)
				r.addNode2Queue(nextVertex)
			}
		}
	}
	return nil
}

func (r *recursiveCrawler) processNode(node IDepNode) ([]IDepNode, error) {
	var depNodes []IDepNode
	var err error
	pkg, ok := node.(*PkgNode)
	if ok {
		depNodes, err = r.downloadAndProcessPkg(pkg)
		if err != nil {
			return depNodes, err
		}

	} else {
		log.Debugf("Found Non Pkg Node not handeled while crawling")
	}

	return depNodes, nil
}

func (r *recursiveCrawler) downloadAndProcessPkg(parentNode *PkgNode) ([]IDepNode, error) {
	var depNodes []IDepNode
	baseDir, err := os.MkdirTemp("", "deps-weaver")
	pd := parentNode.pkg.PackageDetails
	if err != nil {
		log.Debugf("Error while creating a temp dir %s", err)
		return depNodes, err
	}
	defer os.RemoveAll(baseDir)

	_, sourcePath, err := r.pkgManager.DownloadAndGetPackageInfo(baseDir, pd.Name, pd.Version)
	if err != nil {
		log.Debugf("Error while downloading packages %s", err)
		return depNodes, err
	}

	manifestAbsPath, pkgDetails, err := pypi.ParsePythonWheelDist(sourcePath)
	if err != nil {
		log.Debugf("Error while processing package %s", err)
		return depNodes, err
	}

	mani := models.Manifest{Path: manifestAbsPath,
		DisplayPath: manifestAbsPath,
		Ecosystem:   string(pd.Ecosystem)}

	for _, depPd := range pkgDetails {
		pkg := models.Package{PackageDetails: models.PackageDetails(depPd), Manifest: &mani}
		depNodes = append(depNodes, &PkgNode{pkg: &pkg, depth: parentNode.GetDepth() + 1})
	}
	return depNodes, nil
}
