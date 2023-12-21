package vet

import (
	"os"
	"path/filepath"

	"github.com/google/go-github/v54/github"
	"github.com/safedep/cofe/pkg/vet/auth"
	"github.com/safedep/dry/utils"
	"github.com/safedep/vet/pkg/analyzer"
	"github.com/safedep/vet/pkg/common/logger"
	"github.com/safedep/vet/pkg/models"
	"github.com/safedep/vet/pkg/readers"
	"github.com/safedep/vet/pkg/reporter"
	"github.com/safedep/vet/pkg/scanner"
)

type VetInput struct {
	Lockfiles                   []string
	LockfileAs                  string
	BaseDirectory               string
	PurlSpec                    string
	GithubRepoUrls              []string
	GithubOrgUrl                string
	GithubOrgMaxRepositories    int
	ScanExclude                 []string
	TransitiveAnalysis          bool
	TransitiveDepth             int
	Concurrency                 int
	CelFilterExpression         string
	CelFilterSuiteFile          string
	CelFilterFailOnMatch        bool
	JsonReportPath              string
	SilentScan                  bool
	DisableAuthVerifyBeforeScan bool
	ListExperimentalParsers     bool
	IndexUrls                   []string
}

type VetScanner struct {
	input *VetInput
}

func NewVetScanner(input *VetInput) *VetScanner {
	if input.Concurrency == 0 {
		input.Concurrency = 2
	}
	return &VetScanner{input: input}
}

func (v *VetScanner) StartScan() (*VetReport, error) {
	r, err := v.internalStartScan()
	if err != nil {
		logger.Debugf("Failed while running vet to find dependencies.. %s", err)
	}

	return r, err
}

func (v *VetScanner) internalStartScan() (*VetReport, error) {
	readerList := []readers.PackageManifestReader{}
	var reader readers.PackageManifestReader
	var err error

	githubClientBuilder := func() *github.Client {
		githubClient, err := GetGithubClient()
		if err != nil {
			logger.Fatalf("Failed to build Github client: %v", err)
		}

		return githubClient
	}

	baseProjectDir := ""
	// We can easily support both directory and lockfile reader. But current UX
	// contract is to support one of them at a time. Lets not break the contract
	// for now and figure out UX improvement later
	if len(v.input.Lockfiles) > 0 {
		// nolint:ineffassign,staticcheck
		reader, err = readers.NewLockfileReader(v.input.Lockfiles, v.input.LockfileAs)
	} else if len(v.input.GithubRepoUrls) > 0 {
		githubClient := githubClientBuilder()

		// nolint:ineffassign,staticcheck
		reader, err = readers.NewGithubReader(githubClient, v.input.GithubRepoUrls, v.input.LockfileAs)
	} else if len(v.input.GithubOrgUrl) > 0 {
		githubClient := githubClientBuilder()

		// nolint:ineffassign,staticcheck
		reader, err = readers.NewGithubOrgReader(githubClient, &readers.GithubOrgReaderConfig{
			OrganizationURL: v.input.GithubOrgUrl,
			IncludeArchived: false,
			MaxRepositories: v.input.GithubOrgMaxRepositories,
		})
	} else if len(v.input.PurlSpec) > 0 {
		// nolint:ineffassign,staticcheck
		reader, err = readers.NewPurlReader(v.input.PurlSpec)
	} else {
		// nolint:ineffassign,staticcheck
		baseProjectDir, err = filepath.Abs(v.input.BaseDirectory)
		if err != nil {
			return nil, err
		}
		reader, err = readers.NewDirectoryReader(v.input.BaseDirectory, v.input.ScanExclude)
	}

	if err != nil {
		return nil, err
	}

	readerList = append(readerList, reader)

	analyzers := []analyzer.Analyzer{}
	// if !utils.IsEmptyString(v.input.CelFilterExpression) {
	// 	task, err := analyzer.NewCelFilterAnalyzer(v.input.CelFilterExpression,
	// 		v.input.CelFilterFailOnMatch)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	analyzers = append(analyzers, task)
	// }

	// if !utils.IsEmptyString(v.input.CelFilterSuiteFile) {
	// 	task, err := analyzer.NewCelFilterSuiteAnalyzer(v.input.CelFilterSuiteFile,
	// 		v.input.CelFilterFailOnMatch)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	analyzers = append(analyzers, task)
	// }

	reporters := []reporter.Reporter{}
	// if v.input.ConsoleReport {
	// rp, err := reporter.NewConsoleReporter()
	// if err != nil {
	// 	return nil, err
	// }

	// reporters = append(reporters, rp)
	// }

	// if v.input.SummaryReport {
	// 	rp, err := reporter.NewSummaryReporter(reporter.SummaryReporterConfig{
	// 		MaxAdvice: v.input.SummaryReportMaxAdvice,
	// 	})

	// 	if err != nil {
	// 		return err
	// 	}

	// 	reporters = append(reporters, rp)
	// }

	// Trick to create json report structure
	tmpFile, err := os.CreateTemp("", "deps-weaver-vet-json-tmp-")
	if err != nil {
		return nil, err
	}

	defer os.Remove(tmpFile.Name())

	jsonReport, err := reporter.NewJsonReportGenerator(reporter.JsonReportingConfig{
		Path: tmpFile.Name(),
	})
	if err != nil {
		return nil, err
	}
	reporters = append(reporters, jsonReport)

	if !utils.IsEmptyString(v.input.JsonReportPath) {
		rp, err := reporter.NewJsonReportGenerator(reporter.JsonReportingConfig{
			Path: v.input.JsonReportPath,
		})
		if err != nil {
			return nil, err
		}

		reporters = append(reporters, rp)
	}

	//DO not need insights right now.
	insightsEnricher, err := scanner.NewInsightBasedPackageEnricher(scanner.InsightsBasedPackageMetaEnricherConfig{
		ApiUrl:     auth.ApiUrl(),
		ApiAuthKey: auth.ApiKey(),
	})

	if err != nil {
		logger.Debugf("Error while getting Vet Keys... %s", err)
		return nil, err
	}

	enrichers := []scanner.PackageMetaEnricher{
		insightsEnricher,
	}

	pmScanner := scanner.NewPackageManifestScanner(scanner.Config{
		TransitiveAnalysis: v.input.TransitiveAnalysis,
		TransitiveDepth:    v.input.TransitiveDepth,
		ConcurrentAnalyzer: v.input.Concurrency,
		ExcludePatterns:    v.input.ScanExclude,
	}, readerList, enrichers, analyzers, reporters)

	// Trackers to handle UI
	// var packageManifestTracker any
	// var packageTracker any

	vetReport := NewVetReport(baseProjectDir)
	manifestsCount := 0
	pmScanner.WithCallbacks(scanner.ScannerCallbacks{
		OnStartEnumerateManifest: func() {
			logger.Debugf("Starting to enumerate manifests")
		},
		OnEnumerateManifest: func(manifest *models.PackageManifest) {
			logger.Debugf("Discovered a manifest at %s with %d packages",
				manifest.GetDisplayPath(), manifest.GetPackagesCount())
			// ui.IncrementTrackerTotal(packageManifestTracker, 1)
			// ui.IncrementTrackerTotal(packageTracker, int64(manifest.GetPackagesCount()))

			manifestsCount = manifestsCount + 1
			// ui.SetPinnedMessageOnProgressWriter(fmt.Sprintf("Scanning %d discovered manifest(s)",
			// manifestsCount))
		},
		OnStart: func() {
			logger.Debugf("Starting Scan...")
			if !v.input.SilentScan {
				// ui.StartProgressWriter()
			}

			// packageManifestTracker = ui.TrackProgress("Scanning manifests", 0)
			// packageTracker = ui.TrackProgress("Scanning packages", 0)
		},
		OnAddTransitivePackage: func(pkg *models.Package) {
			logger.Debugf("Adding Transitive Package...%v", pkg)
			// ui.IncrementTrackerTotal(packageTracker, 1)
		},
		OnDoneManifest: func(manifest *models.PackageManifest) {
			logger.Debugf("Done Manifest...%v", manifest)
			// ui.IncrementProgress(packageManifestTracker, 1)
			vetReport.AddVetManifest(manifest)
		},
		OnDonePackage: func(pkg *models.Package) {
			logger.Debugf("Done Package...%v", pkg)
			// ui.IncrementProgress(packageTracker, 1)
		},
		BeforeFinish: func() {
			logger.Debugf("Done Scan...")
			// ui.MarkTrackerAsDone(packageManifestTracker)
			// ui.MarkTrackerAsDone(packageTracker)
			// ui.StopProgressWriter()
		},
	})

	logger.Debugf("Starting Package Scanner using Vet... ")
	err = pmScanner.Start()
	if err != nil {
		return nil, err
	}

	vetReport.Print()
	return vetReport, nil
}
