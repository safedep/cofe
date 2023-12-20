package builder

import (
	"errors"
	"os"
	"path"

	G "github.com/dominikbraun/graph"

	"github.com/google/osv-scanner/pkg/lockfile"
	"github.com/safedep/cofe/pkg/core/models"
	"github.com/safedep/cofe/pkg/vet"
	"github.com/safedep/vet/pkg/common/logger"
)

type DepsCrawler struct {
	vetScanner       *vet.VetScanner
	rootpkgGraphNode *pkgGraphNode
	maxDepth         int
	sourcePath       string
	indexUrls        []string
}

type recursiveCrawler struct {
	graph        *iDepNodeGraph
	visitedNodes map[string]iDepNode
	queue        []iDepNode
	index        int
	maxDepth     int
	pkgAnalyzer  *packageAnalyzer
}

func (d *DepsCrawler) newRecursiveCrawler(graph *iDepNodeGraph) (*recursiveCrawler, error) {
	pkgAnalyzer, err := newPackageAnalyzer(d.indexUrls)
	if err != nil {
		return nil, err
	}
	return &recursiveCrawler{
		graph:        graph,
		visitedNodes: make(map[string]iDepNode, 0),
		maxDepth:     d.maxDepth,
		pkgAnalyzer:  pkgAnalyzer,
	}, nil
}

func NewDepsCrawler(vi *vet.VetInput) *DepsCrawler {
	vetScanner := vet.NewVetScanner(vi)
	rootNode := createRootPackage(vi)
	crawler := DepsCrawler{vetScanner: vetScanner,
		rootpkgGraphNode: rootNode,
		maxDepth:         vi.TransitiveDepth,
		sourcePath:       vi.BaseDirectory,
		indexUrls:        vi.IndexUrls}
	return &crawler
}

func createRootPackage(vi *vet.VetInput) *pkgGraphNode {
	mani := models.Manifest{Path: vi.BaseDirectory,
		DisplayPath: vi.BaseDirectory,
		Ecosystem:   string(lockfile.PipEcosystem)}

	name := path.Base(vi.BaseDirectory)
	if name == "" {
		name = "root"
	}
	pd := models.PackageDetails{Name: name}
	return &pkgGraphNode{pkg: *models.NewPackage(&pd, &mani), depth: 0}
}

func (c *DepsCrawler) Crawl() (*graphResult, error) {
	var graph *iDepNodeGraph
	var err error
	graph = newIDepNodeGraph(G.New[string](iDepNodeHashFunc, G.Directed()))
	recCrawler, err := c.newRecursiveCrawler(graph)
	if err != nil {
		logger.Debugf("Error will creating internal crawler %s", err)
		return nil, err
	}
	err = graph.g.AddVertex(c.rootpkgGraphNode)
	if errors.Is(err, G.ErrVertexAlreadyExists) {
		logger.Debugf("Error Root Node Already Exists...")
		return nil, err
	}

	// recCrawler.addNode2Queue(c.rootpkgGraphNode)

	modules := recCrawler.pkgAnalyzer.extractImportedModules(c.sourcePath)
	c.rootpkgGraphNode.pkg.AddImportedModules(modules)
	expModules, err := recCrawler.pkgAnalyzer.extractExportedModules(c.sourcePath)
	if err != nil {
		return nil, err
	}
	c.rootpkgGraphNode.pkg.AddExportedModules(expModules)

	// Scan the base Project to find dependencies
	vetReport, err := c.vetScanner.StartScan()
	if err != nil {
		logger.Debugf("Error while extracting dependencies from base project via vet %s", err)
		return nil, err
	}

	pkgs := vetReport.GetPackages()
	for _, pkg := range pkgs.GetPackages() {
		n := &pkgGraphNode{pkg: *pkg, depth: c.rootpkgGraphNode.GetDepth() + 1}
		err := graph.g.AddVertex(n)
		if err != nil {
			logger.Debugf("Error while adding vertex %s", err)
		} else {
			recCrawler.addNode2Queue(n)
		}
		graph.g.AddEdge(c.rootpkgGraphNode.Key(), n.Key(), G.EdgeData(NewPkgGraphEdgeData()))
	}

	//Do the recursive crawling based on the seed nodes
	recCrawler.crawl()

	gres := newGraphResult(c.rootpkgGraphNode, graph)
	gres.addEdgeWeightBasedOnVulnScore(graph)

	gres.getIDepNodeGraph(true)

	return gres, nil
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

func (r *recursiveCrawler) crawl() error {
	for {
		size, _ := r.graph.g.Size()
		logger.Debugf("Queue Length %d, Edges %d", len(r.queue), size)
		vertex := r.nextVertexFromQueue()
		if vertex == nil {
			break
		}
		// fmt.Printf("Picked up node %s for processing..\n", vertex.Key())
		// r.markVisited(vertex)
		nodes, err := r.processNode(vertex)
		if err != nil {
			logger.Debugf("\tError processing node %s..", vertex.Key())
		}
		for _, n := range nodes {
			err := r.graph.g.AddVertex(n)
			if err != nil {
				if !errors.Is(err, G.ErrVertexAlreadyExists) {
					logger.Debugf("Error while adding vertex %s", err)
				}
			} else {
				// fmt.Printf("\tAdding node %s to the queue..\n", n.Key())
				r.addNode2Queue(n)
			}
			r.graph.g.AddEdge(vertex.Key(), n.Key(), G.EdgeData(NewPkgGraphEdgeData()))

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
		logger.Debugf("Found Non Pkg Node not handeled while crawling")
	}

	return depNodes, nil
}

func (r *recursiveCrawler) downloadAndProcessPkg(parentNode *pkgGraphNode) ([]iDepNode, error) {
	var depNodes []iDepNode
	baseDir, err := os.MkdirTemp("", "deps-weaver")
	if err != nil {
		logger.Debugf("Error while creating a temp dir %s", err)
		return depNodes, err
	}
	defer os.RemoveAll(baseDir)

	packages, err := r.pkgAnalyzer.extractPackagesFromManifest(baseDir, &parentNode.pkg)
	if err != nil {
		logger.Debugf("Error while downloading and processing the pkg %s", err)
		return depNodes, err
	}

	for _, pkg := range packages {
		depNode := &pkgGraphNode{pkg: pkg, depth: parentNode.GetDepth() + 1}
		depNodes = append(depNodes, depNode)
	}
	return depNodes, nil
}
