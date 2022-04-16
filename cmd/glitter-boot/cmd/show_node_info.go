package cmd

import (
	glitterboot "github.com/glitternetwork/glitter-boot"
	"github.com/spf13/cobra"
)

var showNodeInfoCmd = &cobra.Command{
	Use:     "show-node-info",
	Aliases: []string{"show-node-info", "show_node_info"},
	Short:   "show node info",
	Run: func(cmd *cobra.Command, args []string) {
		glitterboot.NodeOperate(cmd.Context(), glitterboot.NodeOpsArgs{
			Type: glitterboot.OpsShowNodeInfo,
		})
	},
}

func init() {
	rootCmd.AddCommand(showNodeInfoCmd)
}
