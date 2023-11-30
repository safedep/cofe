package vet

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/safedep/vet/pkg/models"
)

type VetReport struct {
	baseDir   string
	manifests []*models.PackageManifest
}

func NewVetReport(baseDir string) *VetReport {
	return &VetReport{baseDir: baseDir}
}

func (v *VetReport) AddManifest(man *models.PackageManifest) {
	// Currently vet reports absolute path, make it relative
	man.Path = v.relativePath(v.baseDir, man.Path)
	v.manifests = append(v.manifests, man)
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
	for _, man := range v.manifests {
		fmt.Printf("Manifest %s\n", man.Path)
		for _, p := range man.Packages {
			fmt.Printf("\t Package %v\n", p)
		}
	}
}
