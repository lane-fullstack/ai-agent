package cmd

import (
	"ai-agent/internal/provider"
	"log"

	"ai-agent/internal/config"
	"ai-agent/internal/db"
	"ai-agent/internal/scheduler"
	"ai-agent/internal/telegram"

	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the task scheduler",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.Load()

		database, err := db.Init(config.AsString(cfg["DBPath"]))
		if err != nil {
			log.Fatal(err)
		}

		bot := telegram.NewBot(cfg)

		sched := scheduler.New(database, bot)
		// 初始化 provider
		llm, err := provider.NewGeminiProvider(config.AsString(cfg["GeminiAPIKey"]))
		if err != nil {
			log.Fatal(err)
		}
		provider.L = llm
		err = sched.LoadTasks(llm)
		if err != nil {
			log.Fatal(err)
		}

		sched.Start()

		go telegram.StartListener(bot, sched)

		log.Println("Task manager started")

		select {}
	},
}

func init() {

	rootCmd.AddCommand(startCmd)
}
