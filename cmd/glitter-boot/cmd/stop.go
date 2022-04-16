package cmd

import (
	glitterboot "github.com/glitternetwork/glitter-boot"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "stop glitter and tendermint services",
	Run: func(cmd *cobra.Command, args []string) {
		glitterboot.NodeOperate(cmd.Context(), glitterboot.NodeOpsArgs{
			Type: glitterboot.OpsStopNode,
		})
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
