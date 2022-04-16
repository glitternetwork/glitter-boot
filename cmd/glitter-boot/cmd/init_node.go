package cmd

import (
	glitterboot "github.com/glitternetwork/glitter-boot"
	"github.com/spf13/cobra"
)

var initNodeCmd = &cobra.Command{
	Use:   "init",
	Short: "init node",
	Run: func(cmd *cobra.Command, args []string) {
		glitterboot.NodeOperate(cmd.Context(), initNodeArgs)
	},
}

var initNodeArgs = glitterboot.NodeOpsArgs{}

func init() {
	f := initNodeCmd.PersistentFlags()
	f.StringVarP(&initNodeArgs.Seeds, "seeds", "", "", "Seeds split by ',' example(2e73e0491df978d11f3d928a36b635a4e94ef927@192.167.10.2:26656)")
	f.StringVarP(&initNodeArgs.Moniker, "moniker", "", "", "Moniker for node")
	f.StringVarP(&initNodeArgs.IndexMode, "indexer", "", "es", "IndexMode 'es' or 'kv'")

	f.StringVarP(&initNodeArgs.GlitterBinaryURL, "glitter_bin_url", "", glitterBinURL, "Glitter Binary URL")
	f.StringVarP(&initNodeArgs.TendermintBinaryURL, "tendermint_bin_url", "", tendermintBinURL, "Tendermint Binary URL")
	initNodeArgs.Type = glitterboot.OpsInit

	initNodeCmd.MarkFlagRequired("seeds")
	initNodeCmd.MarkFlagRequired("moniker")
	rootCmd.AddCommand(initNodeCmd)
}
