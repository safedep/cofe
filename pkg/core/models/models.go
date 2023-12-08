package models

import (
	"math"
	"strings"

	"github.com/google/osv-scanner/pkg/lockfile"
	gocvss20 "github.com/pandatix/go-cvss/20"
	gocvss30 "github.com/pandatix/go-cvss/30"
	gocvss31 "github.com/pandatix/go-cvss/31"
	"github.com/safedep/vet/gen/insightapi"
	"github.com/safedep/vet/pkg/common/logger"
)

var IMPACT_2_IMPACT_STRING = map[string]string{

	"VULN_RISK_UNKNOWN":  "UNKNOWN",
	"VULN_RISK_LOW":      "LOW",
	"VULN_RISK_MEDIUM":   "MEDIUM",
	"VULN_RISK_HIGH":     "HIGH",
	"VULN_RISK_CRITICAL": "CRITICAL",
}

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
	vulns           []PkgVuln
	maxVulnScore    int
}

type PkgVuln struct {
	Id              string                     `json:"id"`
	Cve             string                     `json:"cve"`
	Aliases         []string                   `json:"aliases"`
	Title           string                     `json:"title"`
	BaseSeverity    PkgVulnSeverity            `json:"severity"`
	OtherSeverities map[string]PkgVulnSeverity `json:"severities"`
}

type PkgVulnSeverity struct {
	Score  int    `json:"score"`
	Impact string `json:"impact"`
	Type   string `json:"type"`
	Vector string `json:"vector"`
}

func (p *Package) GetMaxVulnScore() int {
	return p.maxVulnScore
}

func (p *Package) GetVulns() []PkgVuln {
	return p.vulns
}

func (p *Package) AddImportedModules(modules []string) {
	if p.importedModules == nil {
		p.importedModules = make(map[string]bool)
	}
	for _, m := range modules {
		p.importedModules[m] = true
	}
}

func (p *Package) AddVulnerabilities(vulns *[]insightapi.PackageVulnerability) {
	if vulns == nil {
		return
	}
	for _, vuln := range *vulns {
		title := ""
		if vuln.Summary != nil {
			title = *vuln.Summary
		}
		if *vuln.Id != "" {
			baseSev, pkgSevs := p.convertSeverity(&vuln)
			cve := p.extractCveFromVulnAliases(*vuln.Aliases)
			pkgVuln := PkgVuln{Id: *vuln.Id,
				Title:           title,
				Cve:             cve,
				Aliases:         *vuln.Aliases,
				BaseSeverity:    *baseSev,
				OtherSeverities: pkgSevs}
			p.maxVulnScore = int(math.Max(float64(p.maxVulnScore), float64(pkgVuln.BaseSeverity.Score)))
			p.vulns = append(p.vulns, pkgVuln)

		} else {
			logger.Debugf("Found vuln with empty title %s", *vuln.Summary)
		}
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

func (c *Package) extractCveFromVulnAliases(aliases []string) string {
	cve := ""
	for _, v := range aliases {
		if strings.HasPrefix(v, "CVE-") || strings.HasPrefix(v, "cve-") {
			cve = v
		}
	}
	return cve
}

func (c *Package) convertSeverity(vuln *insightapi.PackageVulnerability) (*PkgVulnSeverity, map[string]PkgVulnSeverity) {
	basePkgSev := &PkgVulnSeverity{Score: -1.0, Type: "NA", Impact: "Unknown"}
	pkgSev := make(map[string]PkgVulnSeverity, 1)
	for _, sev := range *vuln.Severities {
		score, err := c.parseCVSS(*sev.Score)
		if err != nil {
			logger.Debugf("Error while parsing severity vector %s", err)
			score = -1.0
		}

		impact := c.parseImact(string(*sev.Risk))
		if *sev.Type == insightapi.PackageVulnerabilitySeveritiesTypeCVSSV3 {
			pkgSev["cvss3"] = PkgVulnSeverity{Score: score,
				Type: "cvss3", Impact: impact, Vector: *sev.Score}
		} else if *sev.Type == insightapi.PackageVulnerabilitySeveritiesTypeCVSSV2 {
			pkgSev["cvss2"] = PkgVulnSeverity{Score: score,
				Type: "cvss2", Impact: impact, Vector: *sev.Score}
		} else {
			pkgSev["unknown"] = PkgVulnSeverity{Score: score,
				Type: "unknown", Impact: impact, Vector: *sev.Score}
		}
	}

	if psev, ok := pkgSev["cvss3"]; ok {
		basePkgSev = &psev
	} else if psev, ok := pkgSev["cvss2"]; ok {
		basePkgSev = &psev
	} else if psev, ok := pkgSev["unknown"]; ok {
		basePkgSev = &psev
	}

	return basePkgSev, pkgSev
}

func (c *Package) parseImact(impact string) string {
	i, ok := IMPACT_2_IMPACT_STRING[impact]
	if ok {
		return i
	} else {
		return "UNKNOWN"
	}
}

func (c *Package) parseCVSS(vector string) (int, error) {
	var score float64
	var err error
	switch {
	default: // Should be CVSS v2.0 or is invalid
		cvss, err := gocvss20.ParseVector(vector)
		if err != nil {
			logger.Debugf("Error while parsing cvss vector %s", err)
			return 0.0, err
		}
		score = cvss.BaseScore()
	case strings.HasPrefix(vector, "CVSS:3.0"):
		cvss, err := gocvss30.ParseVector(vector)
		if err != nil {
			logger.Debugf("Error while parsing cvss vector %s", err)
			return 0.0, err
		}
		score = cvss.BaseScore()
	case strings.HasPrefix(vector, "CVSS:3.1"):
		cvss, err := gocvss31.ParseVector(vector)
		if err != nil {
			logger.Debugf("Error while parsing cvss vector %s", err)
			return 0.0, err
		}
		score = cvss.BaseScore()
	}

	return int(math.Round(score)), err
}
