package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/nylas/cli/pkg/util"
	"github.com/spf13/cobra"
)

// CLI root
var root = &cobra.Command{
	Use:   "nylas",
	Short: "The official CLI for Nylas",
	Long:  "The official CLI for Nylas.\n\nBefore using the CLI, you'll need to set your Nylas API Key:\n    $ nylas init",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if _, ok := util.RegionConfig[region]; !ok {
			fmt.Printf("Invalid region '%s'. Valid regions: %s\n", region, strings.Join(validRegions(), ", "))
			os.Exit(1)
		}
	},
}

var region string
var verbose bool

func Execute() {
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	defaultRegion := os.Getenv("NYLAS_REGION")
	if defaultRegion == "" {
		defaultRegion = "us"
	}
	defaultRegion = strings.ToLower(defaultRegion)

	root.PersistentFlags().StringVar(&region, "region", defaultRegion, "Nylas region to use (e.g., 'us' or 'eu'). Can also set NYLAS_REGION env var")
	root.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable verbose debug logging")
}

func validRegions() []string {
	regions := make([]string, 0, len(util.RegionConfig))
	for k := range util.RegionConfig {
		regions = append(regions, k)
	}
	return regions
}

func debugf(format string, a ...interface{}) {
	if verbose {
		fmt.Printf("[debug] "+format+"\n", a...)
	}
}
