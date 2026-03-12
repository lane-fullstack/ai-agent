package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the task scheduler",
	Run: func(cmd *cobra.Command, args []string) {
		// Implement stop logic here, e.g., sending a signal to the running process
		// For simplicity, we'll just print a message.
		// In a real application, you might use a PID file or a control socket.
		fmt.Println("Stop command not implemented yet. Use Ctrl+C to stop the running process.")
		os.Exit(0)
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
