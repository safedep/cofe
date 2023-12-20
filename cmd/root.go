package cmd

import (
	"io"
	"os"

	"github.com/safedep/dry/utils"
	"github.com/safedep/vet/pkg/common/logger"
	"github.com/spf13/cobra"
)

var (
	verbose bool
	debug   bool
	logFile string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cofe",
	Short: "Safedep/Vet on Steroids, prioritize dependencies to upgrade using reachability path",
	Long:  `Safedep/Vet on Steroids, prioritize dependencies to upgrade using reachability path`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Show verbose logs")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Show debug logs")
	rootCmd.PersistentFlags().StringVarP(&logFile, "log", "l", "", "Write command logs to file, use - as for stdout")
}

func initLogger() {
	logger.SetLogLevel(verbose, debug)
	redirectLogToFile(logFile)
}

// Redirect to file or discard log if empty
func redirectLogToFile(path string) {
	// logger.Debugf("Redirecting logger output to: %s", path)
	if !utils.IsEmptyString(path) {
		if path == "-" {
			logger.MigrateTo(os.Stdout)
		} else {
			logger.LogToFile(path)
		}
	} else {
		logger.MigrateTo(io.Discard)
	}
}
