package cmd

import (
	"flag"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	_ "github.com/golang/glog"
)

const Version = "development"

// flags
var (
	identity = "jbod.csi.driver.min.io"
)

var driverCmd = &cobra.Command{
	Use:   os.Args[0],
	Short: "CSI driver for JBODs",
	Long: `
This driver presents a bunch of drives as volumes to containers requesting it.

A bunch of drives can be representing using glob notation. For eg.

1. /mnt/drive{1...32}/path

  This presents 32 drives, whose subdirectory (./path) is the root directory for the CSI driver to operate

2. /mnt/drive{1...32}/path{1...4}

  This presents 32 drives, whose subdirectories path1, path2, path3, path4 are provided as root directories for the CSI driver. This driver will behave as if it was operating with 128 drives (32 * 4)

The driver carves out a unique volume for a particular container from this path by creating a sub-directory. The volume is identified by the subdirectory name. It employs a simple round-robin approach to provisioning from each of the drives given to it.
`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(c *cobra.Command, args []string) error {
		return driver(args)
	},
	Version: Version,
}

func init() {
	viper.AutomaticEnv()
	// parse the go default flagset to get flags for glog and other packages in future
	driverCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	// defaulting this to true so that logs are printed to console
	flag.Set("logtostderr", "true")

	driverCmd.PersistentFlags().StringVarP(&identity, "identity", "i", identity, "identity of this jbod csi driver")

	//suppress the incorrect prefix in glog output
	flag.CommandLine.Parse([]string{})
	viper.BindPFlags(driverCmd.PersistentFlags())

}

func Execute() error {
	return driverCmd.Execute()
}
