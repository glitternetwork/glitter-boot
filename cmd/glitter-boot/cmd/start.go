package cmd

import (
	"fmt"

	glitterboot "github.com/glitternetwork/glitter-boot"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "start [target: `fullnode` or `validator`]",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("please provide a target to start")
			return
		}
		switch args[0] {
		case "fullnode":
			glitterboot.NodeOperate(cmd.Context(), glitterboot.NodeOpsArgs{
				Type: glitterboot.OpsStartFullNode,
			})
		case "validator":
			glitterboot.NodeOperate(cmd.Context(), glitterboot.NodeOpsArgs{
				Type: glitterboot.OpsStartValidator,
			})
		default:
			fmt.Println("must start `fullnode` or `validator`")
		}
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
