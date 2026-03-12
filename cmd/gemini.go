package cmd

import (
	"ai-agent/internal/config"
	"fmt"
	"log"
	"strings"

	"ai-agent/internal/provider"

	"github.com/spf13/cobra"
)

var geminiCmd = &cobra.Command{
	Use:   "gemini",
	Short: "Interact with the Gemini provider",
	Run: func(cmd *cobra.Command, args []string) {
		//apiKey := os.Getenv("GEMINI_API_KEY")
		cfg := config.Load()
		apiKey := cfg.GeminiAPIKey
		if apiKey == "" {
			log.Fatal("Please set GEMINI_API_KEY environment variable")
		}

		p, _ := provider.NewGeminiProvider(apiKey)

		// Example usage:
		// You can extend this to take prompts from command line arguments
		prompt := strings.Join(args, " ")
		if prompt == "" {
			prompt = "Hello, who are you?"
		}

		taskID, _ := cmd.Flags().GetInt64("taskid")
		systemPrompt, _ := cmd.Flags().GetString("system")
		oneShot, _ := cmd.Flags().GetBool("oneshot")

		if systemPrompt != "" {
			p.SetTaskPrompt(taskID, systemPrompt)
			fmt.Printf("System prompt set for task %d.\n", taskID)
		}

		var response string
		if oneShot {
			fmt.Println("Generating one-shot response...")
			response, _ = p.GenerateOneShot(taskID, prompt)
		} else {
			fmt.Println("Chatting...")
			response, _ = p.Chat(taskID, prompt)
		}

		fmt.Println("\n--- Gemini Response ---")
		fmt.Println(response)
	},
}

func init() {
	rootCmd.AddCommand(geminiCmd)
	geminiCmd.Flags().Int64("taskid", 1, "Task ID for the conversation context")
	geminiCmd.Flags().String("system", "", "Set a system prompt for the task")
	geminiCmd.Flags().Bool("oneshot", false, "Use one-shot generation without context")
}
