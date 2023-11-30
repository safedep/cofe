package vet

import (
	"fmt"

	"github.com/google/go-github/v54/github"
	"github.com/safedep/dry/log"
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
	DumpJsonManifestDir         string
	CelFilterExpression         string
	CelFilterSuiteFile          string
	CelFilterFailOnMatch        bool
	MarkdownReportPath          string
	JsonReportPath              string
	ConsoleReport               bool
	SummaryReport               bool
	SummaryReportMaxAdvice      int
	CsvReportPath               string
	SilentScan                  bool
	DisableAuthVerifyBeforeScan bool
	SyncReport                  bool
	SyncReportProject           string
	SyncReportStream            string
	ListExperimentalParsers     bool
}

type VetScanner struct {
	input *VetInput
}

func NewVetScanner(input *VetInput) *VetScanner {
	return &VetScanner{input: input}
}

func (v *VetScanner) StartScan() error {
	err := v.internalStartScan()
	if err != nil {
		log.Debugf("Failed while running vet to find dependencies.. %s", err)
	}

	return err
}

func (v *VetScanner) internalStartScan() error {
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
		reader, err = readers.NewDirectoryReader(v.input.BaseDirectory, v.input.ScanExclude)
	}

	if err != nil {
		return err
	}

	readerList = append(readerList, reader)

	analyzers := []analyzer.Analyzer{}
	if !utils.IsEmptyString(v.input.DumpJsonManifestDir) {
		task, err := analyzer.NewJsonDumperAnalyzer(v.input.DumpJsonManifestDir)
		if err != nil {
			return err
		}

		analyzers = append(analyzers, task)
	}

	if !utils.IsEmptyString(v.input.CelFilterExpression) {
		task, err := analyzer.NewCelFilterAnalyzer(v.input.CelFilterExpression,
			v.input.CelFilterFailOnMatch)
		if err != nil {
			return err
		}

		analyzers = append(analyzers, task)
	}

	if !utils.IsEmptyString(v.input.CelFilterSuiteFile) {
		task, err := analyzer.NewCelFilterSuiteAnalyzer(v.input.CelFilterSuiteFile,
			v.input.CelFilterFailOnMatch)
		if err != nil {
			return err
		}

		analyzers = append(analyzers, task)
	}

	reporters := []reporter.Reporter{}
	if v.input.ConsoleReport {
		rp, err := reporter.NewConsoleReporter()
		if err != nil {
			return err
		}

		reporters = append(reporters, rp)
	}

	if v.input.SummaryReport {
		rp, err := reporter.NewSummaryReporter(reporter.SummaryReporterConfig{
			MaxAdvice: v.input.SummaryReportMaxAdvice,
		})

		if err != nil {
			return err
		}

		reporters = append(reporters, rp)
	}

	if !utils.IsEmptyString(v.input.MarkdownReportPath) {
		rp, err := reporter.NewMarkdownReportGenerator(reporter.MarkdownReportingConfig{
			Path: v.input.MarkdownReportPath,
		})
		if err != nil {
			return err
		}

		reporters = append(reporters, rp)
	}

	if !utils.IsEmptyString(v.input.JsonReportPath) {
		rp, err := reporter.NewJsonReportGenerator(reporter.JsonReportingConfig{
			Path: v.input.JsonReportPath,
		})
		if err != nil {
			return err
		}

		reporters = append(reporters, rp)
	}

	if !utils.IsEmptyString(v.input.CsvReportPath) {
		rp, err := reporter.NewCsvReporter(reporter.CsvReportingConfig{
			Path: v.input.CsvReportPath,
		})
		if err != nil {
			return err
		}

		reporters = append(reporters, rp)
	}

	if v.input.SyncReport {
		rp, err := reporter.NewSyncReporter(reporter.SyncReporterConfig{
			ProjectName: v.input.SyncReportProject,
			StreamName:  v.input.SyncReportStream,
		})
		if err != nil {
			return err
		}

		reporters = append(reporters, rp)
	}

	//DO not need insights right now.
	// insightsEnricher, err := scanner.NewInsightBasedPackageEnricher(scanner.InsightsBasedPackageMetaEnricherConfig{
	// 	ApiUrl:     auth.ApiUrl(),
	// 	ApiAuthKey: auth.ApiKey(),
	// })

	// if err != nil {
	// 	return err
	// }

	enrichers := []scanner.PackageMetaEnricher{
		// insightsEnricher,
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

	manifestsCount := 0
	pmScanner.WithCallbacks(scanner.ScannerCallbacks{
		OnStartEnumerateManifest: func() {
			fmt.Printf("Starting to enumerate manifests")
		},
		OnEnumerateManifest: func(manifest *models.PackageManifest) {
			fmt.Printf("Discovered a manifest at %s with %d packages",
				manifest.GetDisplayPath(), manifest.GetPackagesCount())

			// ui.IncrementTrackerTotal(packageManifestTracker, 1)
			// ui.IncrementTrackerTotal(packageTracker, int64(manifest.GetPackagesCount()))

			manifestsCount = manifestsCount + 1
			// ui.SetPinnedMessageOnProgressWriter(fmt.Sprintf("Scanning %d discovered manifest(s)",
			// manifestsCount))
		},
		OnStart: func() {
			fmt.Printf("Starting Scan...")
			if !v.input.SilentScan {
				// ui.StartProgressWriter()
			}

			// packageManifestTracker = ui.TrackProgress("Scanning manifests", 0)
			// packageTracker = ui.TrackProgress("Scanning packages", 0)
		},
		OnAddTransitivePackage: func(pkg *models.Package) {
			fmt.Printf("Adding Transitive Package...%v", pkg)
			// ui.IncrementTrackerTotal(packageTracker, 1)
		},
		OnDoneManifest: func(manifest *models.PackageManifest) {
			fmt.Printf("Done Manifest...%v", manifest)
			// ui.IncrementProgress(packageManifestTracker, 1)
		},
		OnDonePackage: func(pkg *models.Package) {
			fmt.Printf("Done Package...%v", pkg)
			// ui.IncrementProgress(packageTracker, 1)
		},
		BeforeFinish: func() {
			fmt.Printf("Done Scan...")
			// ui.MarkTrackerAsDone(packageManifestTracker)
			// ui.MarkTrackerAsDone(packageTracker)
			// ui.StopProgressWriter()
		},
	})

	return pmScanner.Start()
}
