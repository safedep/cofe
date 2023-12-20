package builder

import (
	"context"

	"github.com/safedep/codex/pkg/parser/py/imports"
	"github.com/safedep/codex/pkg/utils/py/dir"
	"github.com/safedep/deps_weaver/pkg/core/models"
	"github.com/safedep/deps_weaver/pkg/pm/pypi"
	"github.com/safedep/vet/pkg/common/logger"
)

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
		pd := models.PackageDetails(depPd)
		pkg := *models.NewPackage(&pd, &mani)
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
