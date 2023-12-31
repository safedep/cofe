package vet

import (
	"path/filepath"
	"strings"

	"github.com/safedep/cofe/pkg/core/models"
	"github.com/safedep/vet/pkg/common/logger"
	vet_models "github.com/safedep/vet/pkg/models"
)

type VetReport struct {
	baseDir  string
	packages models.DepPackages
}

func NewVetReport(baseDir string) *VetReport {
	return &VetReport{baseDir: baseDir}
}

func (v *VetReport) AddVetManifest(man *vet_models.PackageManifest) {
	man.Path = v.relativePath(v.baseDir, man.Path)
	manifest := models.Manifest{Path: man.Path,
		DisplayPath: man.DisplayPath,
		Ecosystem:   man.Ecosystem}
	for _, p := range man.Packages {
		pkgDetails := models.PackageDetails{Name: p.Name,
			Version:   p.Version,
			Commit:    p.Commit,
			Ecosystem: p.Ecosystem,
			CompareAs: p.CompareAs}
		pkg := models.Package{PackageDetails: pkgDetails, Manifest: &manifest}
		v.packages.AddPackage(&pkg)
		if p.Insights != nil {
			vulns := p.Insights.Vulnerabilities
			pkg.AddVulnerabilities(vulns)
			pkg.AddScorecard(p.Insights.Scorecard)
		}
	}
}

func (v *VetReport) GetPackages() *models.DepPackages {
	return &v.packages
}

func (v *VetReport) relativePath(basePath, fullPath string) string {
	if basePath == "" {
		return fullPath
	}
	// Clean and normalize the paths to ensure consistency
	basePath = filepath.Clean(basePath)
	fullPath = filepath.Clean(fullPath)

	// Check if the full path is inside the base path
	if !strings.HasPrefix(fullPath, basePath) {
		return fullPath
	}

	// Calculate the relative path
	relativePath := strings.TrimPrefix(fullPath, basePath)
	relativePath = strings.TrimPrefix(relativePath, string(filepath.Separator))

	return relativePath
}

func (v *VetReport) Print() {
	for _, pkg := range v.packages.GetPackages() {
		logger.Infof("Manifest %s Package %s %s\n", pkg.Manifest.Path,
			pkg.PackageDetails.Name, pkg.PackageDetails.Version)
	}
}
