package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	builder "github.com/safedep/cofe/pkg/graph/deps"
	"github.com/safedep/cofe/pkg/pm/pypi"
	"github.com/safedep/cofe/pkg/vet"
	"github.com/safedep/dry/log"
	"github.com/spf13/cobra"
)

var vi vet.VetInput
var export2Graphviz string
var readStdPipConf bool
var export2Csv string

func newScanCommand() *cobra.Command {
	cmd := cobra.Command{
		Use:   "scan",
		Short: "Scan Project & Prioritize Vulnerable and Insecure Dependencies to Upgrade",
		RunE: func(cmd *cobra.Command, args []string) error {
			initLogger()

			if fullpath, err := convertToAbsolutePath(vi.BaseDirectory); err != nil {
				return err
			} else {
				vi.BaseDirectory = fullpath
			}

			iconf := pypi.IndexUrlsConf{ReadStdPipConf: readStdPipConf}
			indexUrls, err := pypi.GetIndexURLs(iconf)
			if err != nil {
				return err
			}
			if len(indexUrls) == 0 {
				return fmt.Errorf("No Index Urls found..")
			}
			vi.IndexUrls = indexUrls
			vi.Concurrency = 3
			crawler := builder.NewDepsCrawler(&vi)
			gres, err := crawler.Crawl()
			if err != nil {
				log.Debugf("Error while running vet %s", err)
				return err
			}
			gres.Print()
			if export2Graphviz != "" {
				gres.Export2Graphviz(fmt.Sprintf("%s.orig.dot", export2Graphviz), false)
			}

			if export2Csv != "" {
				gres.Export2CSV(fmt.Sprintf("%s.orig.csv", export2Csv), false)
			}

			if export2Graphviz != "" {
				gres.Export2Graphviz(export2Graphviz, true)
			}

			if export2Csv != "" {
				gres.Export2CSV(export2Csv, true)
			}

			gres.PrintVulns()
			// gres.PrintLowHygieneNodes()

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
	cmd.Flags().StringVarP(&export2Graphviz, "graphviz", "", "",
		"Export to graphviz")
	cmd.Flags().StringVarP(&export2Csv, "csv", "", "",
		"Export to csv")

	cmd.Flags().BoolVarP(&readStdPipConf, "read-std-conf", "", false,
		"Location of Pip file ")

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
			initLogger()
			iconf := pypi.IndexUrlsConf{ReadStdPipConf: readStdPipConf}
			indexUrls, err := pypi.GetIndexURLs(iconf)
			pm := pypi.NewPypiPackageManager(indexUrls)
			_, baseDir, err := pm.DownloadAndGetPackageInfo(baseDir, pkg, version)
			if err != nil {
				panic(err)
			}
			defer os.RemoveAll(baseDir)
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

	cmd.Flags().BoolVarP(&readStdPipConf, "read-std-conf", "", false,
		"Location of Pip file ")

	return &cmd
}

func init() {
	log.InitZapLogger("Zap")
	rootCmd.AddCommand(newScanCommand())
	rootCmd.AddCommand(newDownloadPypiPkgCommand())
}

// ConvertToAbsolutePath takes a directory path and converts it to an absolute path if it's relative
func convertToAbsolutePath(dir string) (string, error) {
	if filepath.IsAbs(dir) {
		// It's already an absolute path
		return dir, nil
	}
	// Convert to absolute path
	absPath, err := filepath.Abs(dir)
	if err != nil {
		return "", err // return the error
	}
	return absPath, nil
}
