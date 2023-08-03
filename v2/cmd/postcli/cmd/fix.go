package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// fixCmd represents the fix command.
var fixCmd = &cobra.Command{
	Use:   "fix",
	Short: "Fix common issues with PoST data",
	Long: `Postcli fix can be used to fix common issues with PoST data.
For more details take a look at the subcommands.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("fix called")
	},
}

func init() {
	rootCmd.AddCommand(fixCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// fixCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// fixCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
