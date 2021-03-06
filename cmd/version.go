package cmd

import (
	"github.com/ross-weir/rosetta-ergo/pkg/config"
	"github.com/spf13/cobra"
)

var (
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Output rosetta-ergo version",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("rosetta-ergo v%s\n", config.Version)
		},
	}
)
