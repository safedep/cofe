package builder

import (
	"errors"
	"fmt"
	"os"
	"path"

	G "github.com/dominikbraun/graph"
	"github.com/google/osv-scanner/pkg/lockfile"
	"github.com/safedep/deps_weaver/pkg/core/models"
	"github.com/safedep/deps_weaver/pkg/pm/pypi"
	"github.com/safedep/deps_weaver/pkg/vet"
	"github.com/safedep/dry/log"
)

type nodeTypeEnum string

type HashFunc (*func(c iDepNode) string)
type iDepNodeGraph G.Graph[string, iDepNode]

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
	pkg   *models.Package
	depth int
}

func iDepNodeHashFunc(n iDepNode) string {
	return n.Key()
}

// Get the unique key of the node, used for loopups
func (d *pkgGraphNode) Key() string {
	// return fmt.Sprintf("%s:%s:%s", d.pkg.PackageDetails.Name, d.pkg.PackageDetails.Version, d.pkg.PackageDetails.Ecosystem)
	return fmt.Sprintf("%s", d.pkg.PackageDetails.Name)

}

// Get Node type of the node
func (d *pkgGraphNode) GetNodeType() nodeTypeEnum {
	return PackageNodeType
}

// Get Node type of the node
func (d *pkgGraphNode) GetDepth() int {
	return d.depth
}

type DepsCrawler struct {
	vetScanner       *vet.VetScanner
	rootpkgGraphNode *pkgGraphNode
	maxDepth         int
}

type recursiveCrawler struct {
	graph        iDepNodeGraph
	visitedNodes map[string]iDepNode
	queue        []iDepNode
	index        int
	pkgManager   *pypi.PypiPackageManager
	maxDepth     int
}

func (d *DepsCrawler) NewRecursiveCrawler(graph iDepNodeGraph) *recursiveCrawler {
	pm := pypi.NewPypiPackageManager()

	return &recursiveCrawler{
		graph:        graph,
		visitedNodes: make(map[string]iDepNode, 0),
		pkgManager:   pm,
		maxDepth:     d.maxDepth,
	}
}

func (r *recursiveCrawler) isVisited(n iDepNode) bool {
	_, ok := r.visitedNodes[n.Key()]
	return ok
}

func (r *recursiveCrawler) markVisited(n iDepNode) {
	r.visitedNodes[n.Key()] = n
}

func (r *recursiveCrawler) addNode2Queue(n iDepNode) {
	r.queue = append(r.queue, n)
}

func (r *recursiveCrawler) nextVertexFromQueue() iDepNode {
	for r.index < len(r.queue) {
		vertex := r.queue[r.index]
		r.index += 1
		// isVisisted := r.isVisited(vertex)
		isMaxDepth := vertex.GetDepth() > r.maxDepth
		if !isMaxDepth {
			return vertex
		}
	}
	return nil
}

func NewDepsCrawler(vi *vet.VetInput, maxDepth int) *DepsCrawler {
	vetScanner := vet.NewVetScanner(vi)
	rootNode := createRootPackage(vi)
	crawler := DepsCrawler{vetScanner: vetScanner,
		rootpkgGraphNode: rootNode,
		maxDepth:         maxDepth}
	return &crawler
}

func createRootPackage(vi *vet.VetInput) *pkgGraphNode {
	mani := models.Manifest{Path: vi.BaseDirectory,
		DisplayPath: vi.BaseDirectory,
		Ecosystem:   string(lockfile.PipEcosystem)}
	_, name := path.Split(vi.BaseDirectory)
	pd := models.PackageDetails{Name: name}
	return &pkgGraphNode{pkg: &models.Package{PackageDetails: pd, Manifest: &mani}, depth: 0}
}

func (c *DepsCrawler) Crawl() error {
	var graph iDepNodeGraph
	graph = G.New[string](iDepNodeHashFunc, G.Directed())
	recCrawler := c.NewRecursiveCrawler(graph)
	err := graph.AddVertex(c.rootpkgGraphNode)
	if errors.Is(err, G.ErrVertexAlreadyExists) {
		log.Debugf("Error Root Node Already Exists...")
		return err
	}
	recCrawler.addNode2Queue(c.rootpkgGraphNode)

	// Scan the base Project to find dependencies
	vetReport, err := c.vetScanner.StartScan()
	if err != nil {
		log.Debugf("Error while extracting dependencies from base project via vet %s", err)
		return err
	}

	pkgs := vetReport.GetPackages()
	for _, pkg := range pkgs.GetPackages() {
		n := &pkgGraphNode{pkg: pkg, depth: c.rootpkgGraphNode.GetDepth() + 1}
		err := graph.AddVertex(n)
		if err != nil {
			log.Debugf("Error while adding vertex %s", err)
		} else {
			recCrawler.addNode2Queue(n)
		}
		graph.AddEdge(c.rootpkgGraphNode.Key(), n.Key())
	}

	//Do the recursive crawling based on the seed nodes
	recCrawler.crawl()

	// _ = G.DFS(graph, c.rootpkgGraphNode.Key(), func(value string) bool {
	// 	fmt.Println(value)
	// 	return false
	// })

	// file, _ := os.Create("./mygraph.gv")
	// _ = draw.DOT(graph, file)

	// transitiveReduction, err := G.TransitiveReduction(graph)
	// if err != nil {
	// 	log.Debugf("Error in creating transitive graph %s", err)
	// } else {

	// 	file, _ := os.Create("./tans_mygraph.gv")
	// 	_ = draw.DOT(transitiveReduction, file)
	// }

	return nil
}

func (r *recursiveCrawler) crawl() error {
	for {
		size, _ := r.graph.Size()
		log.Debugf("Queue Length %d, Edges %d", len(r.queue), size)
		vertex := r.nextVertexFromQueue()
		if vertex == nil {
			break
		}
		fmt.Printf("Picked up node %s for processing..\n", vertex.Key())
		// r.markVisited(vertex)
		nodes, err := r.processNode(vertex)
		if err != nil {
			log.Debugf("\tError processing node %s..", vertex.Key())
		}
		for _, n := range nodes {
			fmt.Printf("\tAdding node %s to the queue..\n", n.Key())
			err := r.graph.AddVertex(n)
			if err != nil {
				if !errors.Is(err, G.ErrVertexAlreadyExists) {
					log.Debugf("Error while adding vertex %s", err)
				}
			} else {
				r.addNode2Queue(n)
			}
			r.graph.AddEdge(vertex.Key(), n.Key())

		}
	}
	return nil
}

func (r *recursiveCrawler) processNode(node iDepNode) ([]iDepNode, error) {
	var depNodes []iDepNode
	var err error
	pkg, ok := node.(*pkgGraphNode)
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

func (r *recursiveCrawler) downloadAndProcessPkg(parentNode *pkgGraphNode) ([]iDepNode, error) {
	var depNodes []iDepNode
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
		depNodes = append(depNodes, &pkgGraphNode{pkg: &pkg, depth: parentNode.GetDepth() + 1})
	}
	return depNodes, nil
}
