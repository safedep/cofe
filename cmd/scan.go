/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	builder "github.com/safedep/deps_weaver/pkg/graph/deps"
	"github.com/safedep/deps_weaver/pkg/pm/pypi"
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
			crawler := builder.NewDepsCrawler(&vi)
			gres, err := crawler.Crawl()
			if err != nil {
				log.Debugf("Error while running vet %s", err)
				return err
			}
			gres.Print()
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

	cmd.Flags().IntVarP(&vi.TransitiveDepth, "max-depth", "", 2,
		"Depth to analyze transitive dependencies")

	// cmd.Flags().StringArrayVarP(&vi.ScanExclude, "exclude", "", []string{},
	// 	"Name patterns to ignore while scanning a directory")
	// cmd.Flags().StringArrayVarP(&vi.Lockfiles, "lockfiles", "L", []string{},
	// 	"List of lockfiles to scan")
	// cmd.Flags().StringVarP(&vi.PurlSpec, "purl", "", "",
	// 	"PURL to scan")
	// cmd.Flags().StringArrayVarP(&vi.GithubRepoUrls, "github", "", []string{},
	// 	"Github repository URL (Example: https://github.com/{org}/{repo})")
	// cmd.Flags().StringVarP(&vi.GithubOrgUrl, "github-org", "", "",
	// 	"Github organization URL (Example: https://github.com/safedep)")
	// cmd.Flags().IntVarP(&vi.GithubOrgMaxRepositories, "github-org-max-repo", "", 1000,
	// 	"Maximum number of repositories to process for the Github Org")
	// cmd.Flags().StringVarP(&vi.LockfileAs, "lockfile-as", "", "",
	// 	"Parser to use for the lockfile (vet scan parsers to list)")

	// cmd.Flags().BoolVarP(&vi.TransitiveAnalysis, "transitive", "", false,
	// 	"Analyze transitive dependencies")
	// cmd.Flags().IntVarP(&vi.TransitiveDepth, "transitive-depth", "", 2,
	// 	"Analyze transitive dependencies till depth")
	// cmd.Flags().IntVarP(&vi.Concurrency, "concurrency", "C", 5,
	// 	"Number of concurrent analysis to run")
	// cmd.Flags().StringVarP(&vi.CelFilterExpression, "filter", "", "",
	// 	"Filter and print packages using CEL")
	// cmd.Flags().StringVarP(&vi.CelFilterSuiteFile, "filter-suite", "", "",
	// 	"Filter packages using CEL Filter Suite from file")
	// cmd.Flags().BoolVarP(&vi.CelFilterFailOnMatch, "filter-fail", "", false,
	// 	"Fail the scan if the filter match any package (security gate)")
	// cmd.Flags().BoolVarP(&vi.DisableAuthVerifyBeforeScan, "no-verify-auth", "", false,
	// 	"Do not verify auth token before starting scan")
	// cmd.Flags().StringVarP(&vi.JsonReportPath, "report-json", "", "",
	// 	"Generate consolidated JSON report to file (EXPERIMENTAL schema)")

	return &cmd
}

func newDownloadPypiPkgCommand() *cobra.Command {

	var baseDir string
	var pkg string
	var version string

	cmd := cobra.Command{
		Use:   "pypi",
		Short: "Download and extract pypi package",
		RunE: func(cmd *cobra.Command, args []string) error {
			pm := pypi.NewPypiPackageManager()
			_, baseDir, err := pm.DownloadAndGetPackageInfo(baseDir, pkg, version)
			if err != nil {
				panic(err)
			}
			// defer os.RemoveAll(baseDir)
			fmt.Printf("Extracted Package to %s", baseDir)
			_, pkgDetails, err := pypi.ParsePythonWheelDist(baseDir)
			if err != nil {
				panic(err)
			}
			fmt.Println(pkgDetails)
			return nil
		},
	}

	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	cmd.Flags().StringVarP(&baseDir, "dir", "D", wd,
		"Directory to extract")
	cmd.MarkFlagRequired("dir")
	cmd.Flags().StringVarP(&pkg, "pkg", "P", "",
		"Pkg Name ")
	cmd.MarkFlagRequired("pkg")
	cmd.Flags().StringVarP(&version, "version", "V", "",
		"Version")

	return &cmd
}

func init() {
	log.InitZapLogger("Zap")
	rootCmd.AddCommand(newScanCommand())
	rootCmd.AddCommand(newDownloadPypiPkgCommand())
}
