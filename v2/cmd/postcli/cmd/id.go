package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// idCmd represents the id command.
var idCmd = &cobra.Command{
	Use:   "id",
	Short: "Manage your identity for go-spacemesh",
	Long: `id can be used to manage your identity for go-spacemesh.
For more information take a look at the subcommands.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("id called")
	},
}

func init() {
	rootCmd.AddCommand(idCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// idCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// idCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
