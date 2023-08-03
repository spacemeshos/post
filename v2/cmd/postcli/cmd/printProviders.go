package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// printProvidersCmd represents the printProviders command.
var printProvidersCmd = &cobra.Command{
	Use:   "printProviders",
	Short: "Prints the list of available OpenCL providers",
	Long: `Prints the list of available OpenCL providers.
Use the id of the provider to select the device of your choice for initialization.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("printProviders called")
	},
}

func init() {
	initCmd.AddCommand(printProvidersCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// printProvidersCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// printProvidersCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
