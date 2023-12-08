package builder

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	G "github.com/dominikbraun/graph"
	"github.com/dominikbraun/graph/draw"
	"github.com/google/osv-scanner/pkg/lockfile"
	"github.com/safedep/codex/pkg/parser/py/imports"
	"github.com/safedep/codex/pkg/utils/py/dir"
	"github.com/safedep/deps_weaver/pkg/core/models"
	"github.com/safedep/deps_weaver/pkg/pm/pypi"
	"github.com/safedep/deps_weaver/pkg/vet"
	"github.com/safedep/vet/pkg/common/logger"
)

var depth2color = map[int]string{
	0: "#AEEA00", //blue
	1: "#74b9ff", //blue
	2: "#E1BEE7", //yellow
	3: "#E1BEE7", // yellow
	4: "#E1BEE7", // yellow
	5: "#E1BEE7", // yellow

}

var score2Color = map[int]string{
	0:  "#BBDEFB", //blue
	1:  "#BBDEFB", //blue
	2:  "#BBDEFB", //blue
	3:  "#D500F9", // Purple
	4:  "#D500F9", // Purple
	5:  "#D500F9", // Purple
	6:  "#E64A19", // Orange
	7:  "#E64A19", // Orange
	8:  "#E64A19", // Orange
	9:  "#D32F2F", // Orange
	10: "#D32F2F", // Red

}

type graphResult struct {
	rootNode iDepNode
	graph    iDepNodeGraph
	// import2PkgMap map[string][]iDepNode
	export2PkgMap        map[string][]iDepNode
	cachedReachableGraph iDepNodeGraph
}

func newGraphResult(rootNode iDepNode, graph iDepNodeGraph) *graphResult {
	gres := graphResult{rootNode: rootNode, graph: graph,
		export2PkgMap: make(map[string][]iDepNode, 0)}

	gres.addExportedModulesMap()
	return &gres
}

func (g *graphResult) addExportedModulesMap() {
	// Post Processing of the graph
	_ = G.DFS(g.graph, g.rootNode.Key(), func(value string) bool {
		node, err := g.graph.Vertex(value)
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

func (g *graphResult) RemoveEdgesBasedOnImportedModules() error {
	edges, err := g.graph.Edges()
	if err != nil {
		logger.Debugf("Error while getting edges %s", err)
		return err
	}
	for _, edge := range edges {
		sv, _ := g.graph.Vertex(edge.Source)
		tv, _ := g.graph.Vertex(edge.Target)
		targetNode, _ := tv.(*pkgGraphNode)
		sourceNode, _ := sv.(*pkgGraphNode)
		importedModules := sourceNode.pkg.GetImportedModules()
		expotedModules := targetNode.pkg.GetExportedModules()
		// Import and export module match?
		targetPkgName := targetNode.pkg.PackageDetails.Name

		ieMatch := g.stringsIntersect(importedModules, expotedModules)
		targetInImports := g.targetPkgNameInImportedModules(targetPkgName, importedModules)
		if !targetInImports && !ieMatch {
			g.graph.RemoveEdge(sv.Key(), tv.Key())
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

func (g *graphResult) getIDepNodeGraph(reachable bool) (iDepNodeGraph, error) {
	if reachable {
		if g.cachedReachableGraph == nil {
			gg, err := g.ReachableGraph()
			if err != nil {
				logger.Debugf("Error while genenerating reachable graph %s", err)
				return nil, err
			}
			g.cachedReachableGraph = gg
		}
		return g.cachedReachableGraph, nil
	}

	return g.graph, nil
}

// useful to generate csv content
func (g *graphResult) depth2Color(depth int) string {
	c, ok := depth2color[depth]
	if !ok {
		return "#b2bec3" //grey color
	}

	return c
}

func (g *graphResult) score2Color(score int) string {
	c, ok := score2Color[score]
	if !ok {
		return "#b2bec3" //grey color
	}

	return c
}

func (g *graphResult) node2Color(n iDepNode) string {
	pkgNode, _ := n.(*pkgGraphNode)

	if pkgNode.pkg.GetMaxVulnScore() > 7 {
		return g.score2Color(pkgNode.pkg.GetMaxVulnScore())
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
	return draw.DOT(gg, file)
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

func (g *graphResult) exportEdges2CSV(gg iDepNodeGraph, outpath string) error {
	edges, err := gg.Edges()
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
	header := []string{"source", "target", "color", "time"}
	if err := writer.Write(header); err != nil {
		return err
	}

	i := 0
	// Write each record to the CSV file
	for _, edge := range edges {
		sv, _ := g.graph.Vertex(edge.Source)
		sourceNode, _ := sv.(*pkgGraphNode)
		recordRow := []string{
			edge.Source,
			edge.Target,
			g.depth2Color(sourceNode.depth),
			g.depth2timestamp(sourceNode.depth),
			strconv.Itoa(sourceNode.pkg.GetMaxVulnScore()),
		}
		if err := writer.Write(recordRow); err != nil {
			return err
		}
		i += 1
	}

	return nil

}

func (g *graphResult) getUniqueVertices(gg iDepNodeGraph) (*[]iDepNode, error) {
	edges, err := gg.Edges()
	if err != nil {
		logger.Debugf("Error while getting edges %s", err)
		return nil, err
	}

	var vertices []iDepNode
	cache := make(map[string]bool, 0)
	// Write each record to the CSV file
	for _, edge := range edges {
		sv, _ := g.graph.Vertex(edge.Source)
		_, ok := cache[edge.Source]
		if !ok {
			cache[edge.Source] = true
			vertices = append(vertices, sv)
		}

		tv, _ := g.graph.Vertex(edge.Target)
		_, ok = cache[edge.Target]
		if !ok {
			cache[edge.Target] = true
			vertices = append(vertices, tv)
		}
	}

	return &vertices, nil
}

func (g *graphResult) exportMetadata2CSV(gg iDepNodeGraph, outpath string) error {
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
	header := []string{"id", "node_color", "node_value", "type", "max_score"}
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
			strconv.Itoa(v.GetDepth()),
			strconv.Itoa(v.GetDepth()),
			strconv.Itoa(pkgNode.pkg.GetMaxVulnScore()),
		}
		if err := writer.Write(recordRow); err != nil {
			logger.Warnf("Error while metadata writig to CSV file %s", err)
			return err
		}
	}
	return nil

}

func (g *graphResult) Print() {
	fmt.Printf("%s\n", g.rootNode.Key())
	_ = G.BFS(g.graph, g.rootNode.Key(), func(value string) bool {
		node, err := g.graph.Vertex(value)
		if err != nil {
			return false
		}
		node.GetDepth()
		fmt.Printf("%s %s\n", g.spaces(node.GetDepth()), node.Key())
		return false
	})
}

func (g *graphResult) ReachableGraph() (iDepNodeGraph, error) {

	logger.Debugf("Reducing Reachable Graph ...")
	reachableNodes := make(map[string]bool, 0)

	reachableNodes[g.rootNode.Key()] = true
	_ = G.BFS(g.graph, g.rootNode.Key(), func(value string) bool {
		reachableNodes[value] = true
		return false
	})

	gg, err := g.graph.Clone()
	if err != nil {
		return nil, err
	}

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

			// err = gg.RemoveVertex(v)
			// logger.Debugf("Removed vertex again %s", err)
			// _, err = gg.Vertex(v)
			// logger.Debugf("Fetching after deletion %s", err)

		}
	}

	_, err = gg.Vertex("tqdm")
	logger.Debugf("tqdm Fetching after deletion %s", err)
	return gg, nil
}

func (g *graphResult) spaces(n int) string {
	var sb strings.Builder
	for i := 0; i < n; i++ {
		sb.WriteString("..")
	}
	return sb.String()
}

type nodeTypeEnum string
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
	pkg   models.Package
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
	sourcePath       string
	indexUrls        []string
}

type recursiveCrawler struct {
	graph        iDepNodeGraph
	visitedNodes map[string]iDepNode
	queue        []iDepNode
	index        int
	maxDepth     int
	pkgAnalyzer  *packageAnalyzer
}

type packageAnalyzer struct {
	pyCodeParser *imports.CodeParser
	pkgManager   *pypi.PypiPackageManager
}

func newPackageAnalyzer(indexUrls []string) (*packageAnalyzer, error) {
	cf := imports.NewPyCodeParserFactory()
	parser, err := cf.NewCodeParser()
	if err != nil {
		logger.Warnf("Error while creating parser %v", err)
		return nil, err
	}

	pkgManager := pypi.NewPypiPackageManager(indexUrls)

	return &packageAnalyzer{pyCodeParser: parser,
		pkgManager: pkgManager}, nil
}

func (d *DepsCrawler) newRecursiveCrawler(graph iDepNodeGraph) (*recursiveCrawler, error) {
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
	return &pkgGraphNode{pkg: models.Package{PackageDetails: pd, Manifest: &mani}, depth: 0}
}

func (c *DepsCrawler) Crawl() (*graphResult, error) {
	var graph iDepNodeGraph
	var err error
	graph = G.New[string](iDepNodeHashFunc, G.Directed())
	recCrawler, err := c.newRecursiveCrawler(graph)
	if err != nil {
		logger.Debugf("Error will creating internal crawler %s", err)
		return nil, err
	}
	err = graph.AddVertex(c.rootpkgGraphNode)
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
		err := graph.AddVertex(n)
		if err != nil {
			logger.Debugf("Error while adding vertex %s", err)
		} else {
			recCrawler.addNode2Queue(n)
		}
		graph.AddEdge(c.rootpkgGraphNode.Key(), n.Key())
	}

	//Do the recursive crawling based on the seed nodes
	recCrawler.crawl()

	gres := newGraphResult(c.rootpkgGraphNode, graph)
	return gres, nil
}

func (r *recursiveCrawler) crawl() error {
	for {
		size, _ := r.graph.Size()
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
			err := r.graph.AddVertex(n)
			if err != nil {
				if !errors.Is(err, G.ErrVertexAlreadyExists) {
					logger.Debugf("Error while adding vertex %s", err)
				}
			} else {
				// fmt.Printf("\tAdding node %s to the queue..\n", n.Key())
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

func (r *packageAnalyzer) extractPackagesFromManifest(baseDir string,
	parentPkg *models.Package) ([]models.Package, error) {

	pd := parentPkg.PackageDetails
	// Download and process package file
	packages := make([]models.Package, 0)
	_, sourcePath, err := r.pkgManager.DownloadAndGetPackageInfo(baseDir, pd.Name, pd.Version)
	if err != nil {
		logger.Debugf("Error while downloading packages %s", err)
		return packages, err
	}

	manifestAbsPath, pkgDetails, err := pypi.ParsePythonWheelDist(sourcePath)
	if err != nil {
		logger.Debugf("Error while processing package %s", err)
		return packages, err
	}

	parentPkg.AddImportedModules(r.extractImportedModules(sourcePath))
	expModules, _ := r.extractExportedModules(sourcePath)
	if err != nil {
		return packages, err
	}
	parentPkg.AddExportedModules(expModules)

	maniRelPath, _ := dir.RelativePath(sourcePath, manifestAbsPath)
	mani := models.Manifest{Path: maniRelPath,
		DisplayPath: manifestAbsPath,
		Ecosystem:   string(pd.Ecosystem)}

	for _, depPd := range pkgDetails {
		pkg := models.Package{PackageDetails: models.PackageDetails(depPd), Manifest: &mani}
		packages = append(packages, pkg)
	}

	return packages, nil
}

/*
Find all imported modules based on the import statements in the code
*/
func (r *packageAnalyzer) extractImportedModules(sourcePath string) []string {
	ctx := context.Background()
	includeExtensions := []string{".py"}
	excludeDirs := []string{".git", "test"}

	rootPkgs, _ := r.pyCodeParser.FindImportedModules(ctx, sourcePath,
		true, includeExtensions, excludeDirs)
	return rootPkgs.GetPackagesNames()
}

/*
Find all exported modules by package itself that can be imported by others
*/
func (r *packageAnalyzer) extractExportedModules(sourcePath string) ([]string, error) {
	ctx := context.Background()
	logger.Debugf("Finding Exported module at %s", sourcePath)
	exportedModules, err := r.pyCodeParser.FindExportedModules(ctx, sourcePath)
	if err != nil {
		logger.Debugf("Error while finding exported modules %s", err)
		return nil, err
	}
	modules := exportedModules.GetExportedModules()
	rp, _ := dir.FindTopLevelModules(sourcePath)
	logger.Debugf("Found Exported modules  %s %s %s", modules, rp, err)
	return modules, nil
}
