package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// mergeCmd represents the merge command.
var mergeCmd = &cobra.Command{
	Use:   "merge",
	Short: "merge chunks of PoST data",
	Long: `Initialization can be done in chunks.
This command can be used to merge the chunks into a single directory.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("merge called")
	},
}

func init() {
	initCmd.AddCommand(mergeCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// mergeCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// mergeCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
