/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/safedep/deps_weaver/pkg/vet"
	"github.com/safedep/dry/log"
	"github.com/spf13/cobra"
)

var vi vet.VetInput

func newScanCommand() *cobra.Command {
	cmd := cobra.Command{
		Use:   "scan",
		Short: "Scan and analyse package manifests",
		RunE: func(cmd *cobra.Command, args []string) error {
			vetScanner := vet.NewVetScanner(&vi)
			vetScanner.StartScan()
			return nil
		},
	}

	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	cmd.Flags().BoolVarP(&vi.SilentScan, "silent", "s", false,
		"Silent scan to prevent rendering UI")
	cmd.Flags().StringVarP(&vi.BaseDirectory, "directory", "D", wd,
		"The directory to scan for lockfiles")
	cmd.Flags().StringArrayVarP(&vi.ScanExclude, "exclude", "", []string{},
		"Name patterns to ignore while scanning a directory")
	cmd.Flags().StringArrayVarP(&vi.Lockfiles, "lockfiles", "L", []string{},
		"List of lockfiles to scan")
	cmd.Flags().StringVarP(&vi.PurlSpec, "purl", "", "",
		"PURL to scan")
	cmd.Flags().StringArrayVarP(&vi.GithubRepoUrls, "github", "", []string{},
		"Github repository URL (Example: https://github.com/{org}/{repo})")
	cmd.Flags().StringVarP(&vi.GithubOrgUrl, "github-org", "", "",
		"Github organization URL (Example: https://github.com/safedep)")
	cmd.Flags().IntVarP(&vi.GithubOrgMaxRepositories, "github-org-max-repo", "", 1000,
		"Maximum number of repositories to process for the Github Org")
	cmd.Flags().StringVarP(&vi.LockfileAs, "lockfile-as", "", "",
		"Parser to use for the lockfile (vet scan parsers to list)")
	cmd.Flags().BoolVarP(&vi.TransitiveAnalysis, "transitive", "", false,
		"Analyze transitive dependencies")
	cmd.Flags().IntVarP(&vi.TransitiveDepth, "transitive-depth", "", 2,
		"Analyze transitive dependencies till depth")
	cmd.Flags().IntVarP(&vi.Concurrency, "concurrency", "C", 5,
		"Number of concurrent analysis to run")
	cmd.Flags().StringVarP(&vi.DumpJsonManifestDir, "json-dump-dir", "", "",
		"Dump enriched package manifests as JSON files to dir")
	cmd.Flags().StringVarP(&vi.CelFilterExpression, "filter", "", "",
		"Filter and print packages using CEL")
	cmd.Flags().StringVarP(&vi.CelFilterSuiteFile, "filter-suite", "", "",
		"Filter packages using CEL Filter Suite from file")
	cmd.Flags().BoolVarP(&vi.CelFilterFailOnMatch, "filter-fail", "", false,
		"Fail the scan if the filter match any package (security gate)")
	cmd.Flags().BoolVarP(&vi.DisableAuthVerifyBeforeScan, "no-verify-auth", "", false,
		"Do not verify auth token before starting scan")
	cmd.Flags().StringVarP(&vi.MarkdownReportPath, "report-markdown", "", "",
		"Generate consolidated markdown report to file")
	cmd.Flags().BoolVarP(&vi.ConsoleReport, "report-console", "", false,
		"Print a report to the console")
	cmd.Flags().BoolVarP(&vi.SummaryReport, "report-summary", "", true,
		"Print a summary report with actionable advice")
	cmd.Flags().IntVarP(&vi.SummaryReportMaxAdvice, "report-summary-max-advice", "", 5,
		"Maximum number of package risk advice to show")
	cmd.Flags().StringVarP(&vi.CsvReportPath, "report-csv", "", "",
		"Generate CSV report of filtered packages")
	cmd.Flags().StringVarP(&vi.JsonReportPath, "report-json", "", "",
		"Generate consolidated JSON report to file (EXPERIMENTAL schema)")
	cmd.Flags().BoolVarP(&vi.SyncReport, "report-sync", "", false,
		"Enable syncing report data to cloud")
	cmd.Flags().StringVarP(&vi.SyncReportProject, "report-sync-project", "", "",
		"Project name to use in cloud")
	cmd.Flags().StringVarP(&vi.SyncReportStream, "report-sync-stream", "", "",
		"Project stream name (e.g. branch) to use in cloud")

	return &cmd
}

func init() {
	log.InitZapLogger("Zap")
	rootCmd.AddCommand(newScanCommand())
}
