package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ai-agent",
	Short: "AI Agent is a task scheduler and executor",
	Long:  `AI Agent allows you to schedule tasks and execute commands, sending results via Telegram.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Commands are added in their respective init() functions
}
