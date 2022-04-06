package cmd

import (
	glitterboot "github.com/glitternetwork/glitter-boot"
	"github.com/spf13/cobra"
)

var validatorCmd = &cobra.Command{
	Use:   "validator",
	Short: "validator manager",
}

var setupValidatorArgs = glitterboot.SetupNodeArgs{}

func init() {
	f := validatorSetupCmd.PersistentFlags()
	f.StringVarP(&setupValidatorArgs.Seeds, "seeds", "", "", "Seeds split by ',' example(2e73e0491df978d11f3d928a36b635a4e94ef927@192.167.10.2:26656)")
	f.StringVarP(&setupValidatorArgs.Moniker, "moniker", "", "", "Moniker for node")
	f.StringVarP(&setupValidatorArgs.IndexMode, "indexer", "", "es", "IndexMode 'es' or 'kv'")
	f.StringVarP(&setupValidatorArgs.GlitterBinaryURL, "glitter_bin_url", "", "", "Glitter Binary URL")
	f.StringVarP(&setupValidatorArgs.TendermintBinaryURL, "tendermint_bin_url", "", "", "Tendermint Binary URL")
	validatorCmd.MarkFlagRequired("seeds")
	validatorCmd.MarkFlagRequired("moniker")
	validatorCmd.AddCommand(validatorSetupCmd)
	rootCmd.AddCommand(validatorCmd)
}

var validatorSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Setup and run a validator",
	Long:  `Setup and run a validator.`,
	Run: func(cmd *cobra.Command, args []string) {
		glitterboot.SetupNode(cmd.Context(), glitterboot.SetupNodeArgs{
			ValidatorMode:       true,
			Seeds:               setupValidatorArgs.Seeds,
			Moniker:             setupValidatorArgs.Moniker,
			IndexMode:           setupValidatorArgs.IndexMode,
			GlitterBinaryURL:    setupValidatorArgs.GlitterBinaryURL,
			TendermintBinaryURL: setupValidatorArgs.TendermintBinaryURL,
		})
	},
}
