package models

import "github.com/google/osv-scanner/pkg/lockfile"

type PackageDetails lockfile.PackageDetails

type Manifest struct {
	// Filesystem path of this manifest
	Path string `json:"path"`

	// When we scan non-path entities like Github org / repo
	// then only path doesn't make sense, which is more local
	// temporary file path
	DisplayPath string `json:"display_path"`

	// Ecosystem to interpret this manifest
	Ecosystem string `json:"ecosystem"`
}

// Represents a package manifest that contains a list
// of packages. Example: pom.xml, requirements.txt
type Package struct {
	PackageDetails  PackageDetails
	Manifest        *Manifest // Link to Manifest
	exportedModules map[string]bool
	importedModules map[string]bool
}

func (p *Package) AddImportedModules(modules []string) {
	if p.importedModules == nil {
		p.importedModules = make(map[string]bool)
	}
	for _, m := range modules {
		p.importedModules[m] = true
	}
}

func (p *Package) AddExportedModules(modules []string) {
	if p.exportedModules == nil {
		p.exportedModules = make(map[string]bool)
	}
	for _, m := range modules {
		p.exportedModules[m] = true
	}
}

func (p *Package) GetImportedModules() []string {
	var mods []string
	for mod, _ := range p.importedModules {
		mods = append(mods, mod)
	}

	return mods
}

func (p *Package) GetExportedModules() []string {
	var mods []string
	for mod, _ := range p.exportedModules {
		mods = append(mods, mod)
	}

	return mods
}

type DepPackages struct {
	packages []*Package
}

func (p *DepPackages) AddPackage(pkg *Package) {
	p.packages = append(p.packages, pkg)
}

func (p *DepPackages) GetPackages() []*Package {
	return p.packages
}
