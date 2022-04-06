package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "glitter-boot",
	Short: "Glitter bootstrap tool",
	Long: `Glitter bootstrap tool
`,
}

func Execute() {
	//doc.GenMarkdownTree(rootCmd, "./../")
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}

}
