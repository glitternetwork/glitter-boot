package cmd

import (
	glitterboot "github.com/glitternetwork/glitter-boot"
	"github.com/spf13/cobra"
)

var fullnodeCmd = &cobra.Command{
	Use:   "fullnode",
	Short: "fullnode manager",
}

var setupFullNodeArgs = glitterboot.SetupNodeArgs{}

func init() {
	f := fullNodeSetupCmd.PersistentFlags()
	f.StringVarP(&setupFullNodeArgs.Seeds, "seeds", "", "", "Seeds split by ',' example(2e73e0491df978d11f3d928a36b635a4e94ef927@192.167.10.2:26656)")
	f.StringVarP(&setupFullNodeArgs.Moniker, "moniker", "", "", "Moniker for node")
	f.StringVarP(&setupFullNodeArgs.IndexMode, "indexer", "", "es", "IndexMode 'es' or 'kv'")
	f.StringVarP(&setupFullNodeArgs.GlitterBinaryURL, "glitter_bin_url", "", "", "Glitter Binary URL")
	f.StringVarP(&setupFullNodeArgs.TendermintBinaryURL, "tendermint_bin_url", "", "", "Tendermint Binary URL")
	fullnodeCmd.MarkFlagRequired("seeds")
	fullnodeCmd.MarkFlagRequired("moniker")
	fullnodeCmd.AddCommand(validatorSetupCmd)
	rootCmd.AddCommand(fullnodeCmd)
}

var fullNodeSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Setup and run a fullnode",
	Long:  `Setup and run a fullnode.`,
	Run: func(cmd *cobra.Command, args []string) {
		glitterboot.SetupNode(cmd.Context(), glitterboot.SetupNodeArgs{
			ValidatorMode:       false,
			Seeds:               setupValidatorArgs.Seeds,
			Moniker:             setupValidatorArgs.Moniker,
			IndexMode:           setupValidatorArgs.IndexMode,
			GlitterBinaryURL:    setupValidatorArgs.GlitterBinaryURL,
			TendermintBinaryURL: setupValidatorArgs.TendermintBinaryURL,
		})
	},
}
