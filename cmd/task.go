package cmd

import (
	"ai-agent/internal/telegram"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"ai-agent/internal/config"
	"ai-agent/internal/db"
	"ai-agent/internal/executor"
	"ai-agent/internal/model"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(taskCmd)
	taskCmd.AddCommand(listTasksCmd)
	taskCmd.AddCommand(addTaskCmd)
	taskCmd.AddCommand(updateTaskCmd)
	taskCmd.AddCommand(deleteTaskCmd)
	taskCmd.AddCommand(addScraperTaskCmd)
	taskCmd.AddCommand(runTaskCmd)

	addTaskCmd.Flags().String("name", "", "Task name")
	addTaskCmd.Flags().String("type", "", "Task type")
	addTaskCmd.Flags().String("command", "", "Task command")
	addTaskCmd.Flags().String("cron", "", "Cron expression")
	addTaskCmd.Flags().Bool("enabled", true, "Enable task")
	addTaskCmd.Flags().Int64("chat_id", 0, "Chat ID")
	addTaskCmd.Flags().Bool("notify", true, "Notify on completion")

	updateTaskCmd.Flags().Int64("id", 0, "Task ID")
	updateTaskCmd.Flags().String("name", "", "Task name")
	updateTaskCmd.Flags().String("type", "", "Task type")
	updateTaskCmd.Flags().String("command", "", "Task command")
	updateTaskCmd.Flags().String("cron", "", "Cron expression")
	updateTaskCmd.Flags().Bool("enabled", true, "Enable task")
	updateTaskCmd.Flags().Int64("chat_id", 0, "Chat ID")
	updateTaskCmd.Flags().Bool("notify", true, "Notify on completion")

	deleteTaskCmd.Flags().Int64("id", 0, "Task ID")

	addScraperTaskCmd.Flags().String("cron", "@every 1h", "Cron expression")
	addScraperTaskCmd.Flags().Int64("chat_id", 0, "Chat ID")

	runTaskCmd.Flags().Int64("id", 0, "Task ID")
}

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage tasks",
}

var listTasksCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tasks",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.Load()
		database, err := db.Init(config.AsString(cfg["DBPath"]))
		if err != nil {
			log.Fatal(err)
		}

		rows, err := database.Query("SELECT id, name, type, command, cron, enabled, chat_id, notify FROM tasks")
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()

		fmt.Println("Tasks:")
		fmt.Printf("%-5s %-20s %-10s %-20s %-15s %-10s %-15s %-10s\n", "ID", "Name", "Type", "Command", "Cron", "Enabled", "ChatID", "Notify")
		for rows.Next() {
			var t model.Task
			var enabled, notify int
			err = rows.Scan(&t.ID, &t.Name, &t.Type, &t.Command, &t.Cron, &enabled, &t.ChatID, &notify)
			if err != nil {
				log.Fatal(err)
			}
			t.Enabled = enabled == 1
			t.Notify = notify == 1
			fmt.Printf("%-5d %-20s %-10s %-20s %-15s %-10v %-15d %-10v\n", t.ID, t.Name, t.Type, t.Command, t.Cron, t.Enabled, t.ChatID, t.Notify)
		}
	},
}

var addTaskCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new task",
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		typ, _ := cmd.Flags().GetString("type")
		command, _ := cmd.Flags().GetString("command")
		cron, _ := cmd.Flags().GetString("cron")
		enabled, _ := cmd.Flags().GetBool("enabled")
		chatID, _ := cmd.Flags().GetInt64("chat_id")
		notify, _ := cmd.Flags().GetBool("notify")

		if name == "" || typ == "" || command == "" || cron == "" {
			log.Fatal("Missing required flags: name, type, command, cron")
		}

		cfg := config.Load()
		database, err := db.Init(config.AsString(cfg["DBPath"]))
		if err != nil {
			log.Fatal(err)
		}

		_, err = database.Exec("INSERT INTO tasks (name, type, command, cron, enabled, chat_id, notify) VALUES (?, ?, ?, ?, ?, ?, ?)",
			name, typ, command, cron, enabled, chatID, notify)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Task added successfully")
	},
}

var updateTaskCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an existing task",
	Run: func(cmd *cobra.Command, args []string) {
		id, _ := cmd.Flags().GetInt64("id")
		name, _ := cmd.Flags().GetString("name")
		typ, _ := cmd.Flags().GetString("type")
		command, _ := cmd.Flags().GetString("command")
		cron, _ := cmd.Flags().GetString("cron")
		enabled, _ := cmd.Flags().GetBool("enabled")
		chatID, _ := cmd.Flags().GetInt64("chat_id")
		notify, _ := cmd.Flags().GetBool("notify")

		if id == 0 {
			log.Fatal("Missing required flag: id")
		}

		cfg := config.Load()
		database, err := db.Init(config.AsString(cfg["DBPath"]))
		if err != nil {
			log.Fatal(err)
		}

		// Build update query dynamically
		query := "UPDATE tasks SET "
		params := []interface{}{}
		if name != "" {
			query += "name = ?, "
			params = append(params, name)
		}
		if typ != "" {
			query += "type = ?, "
			params = append(params, typ)
		}
		if command != "" {
			query += "command = ?, "
			params = append(params, command)
		}
		if cron != "" {
			query += "cron = ?, "
			params = append(params, cron)
		}

		if cmd.Flags().Changed("enabled") {
			query += "enabled = ?, "
			params = append(params, enabled)
		}
		if cmd.Flags().Changed("chat_id") {
			query += "chat_id = ?, "
			params = append(params, chatID)
		}
		if cmd.Flags().Changed("notify") {
			query += "notify = ?, "
			params = append(params, notify)
		}

		if len(params) == 0 {
			fmt.Println("No fields to update")
			return
		}

		query = query[:len(query)-2]
		query += " WHERE id = ?"
		params = append(params, id)

		_, err = database.Exec(query, params...)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Task updated successfully")
	},
}

var deleteTaskCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a task",
	Run: func(cmd *cobra.Command, args []string) {
		id, _ := cmd.Flags().GetInt64("id")
		if id == 0 {
			log.Fatal("Missing required flag: id")
		}

		cfg := config.Load()
		database, err := db.Init(config.AsString(cfg["DBPath"]))
		if err != nil {
			log.Fatal(err)
		}

		_, err = database.Exec("DELETE FROM tasks WHERE id = ?", id)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Task deleted successfully")
	},
}

var addScraperTaskCmd = &cobra.Command{
	Use:   "add-scraper",
	Short: "Add a task to run the scraper periodically",
	Run: func(cmd *cobra.Command, args []string) {
		cron, _ := cmd.Flags().GetString("cron")
		chatID, _ := cmd.Flags().GetInt64("chat_id")

		if cron == "" {
			log.Fatal("Missing required flag: cron")
		}

		cfg := config.Load()
		database, err := db.Init(config.AsString(cfg["DBPath"]))
		if err != nil {
			log.Fatal(err)
		}

		// Use the new internal task type
		command := "FetchTrumpTruths"

		_, err = database.Exec("INSERT INTO tasks (name, type, command, cron, enabled, chat_id, notify) VALUES (?, ?, ?, ?, ?, ?, ?)",
			"Scrape TrumpTruths", "internal", command, cron, 1, chatID, 1)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Scraper task added successfully using internal Go function.")
	},
}

var runTaskCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a task immediately",
	Run: func(cmd *cobra.Command, args []string) {

		id, _ := cmd.Flags().GetInt64("id")
		if id == 0 {
			log.Fatal("Missing required flag: id")
		}

		cfg := config.Load()
		database, err := db.Init(config.AsString(cfg["DBPath"]))
		if err != nil {
			log.Fatal(err)
		}

		bot := telegram.NewBot(cfg)

		bot.Send(1654278367, "test")

		var t model.Task
		var enabled, notify int
		err = database.QueryRow("SELECT id, name, type, command, cron, enabled, chat_id, notify FROM tasks WHERE id = ?", id).Scan(&t.ID, &t.Name, &t.Type, &t.Command, &t.Cron, &enabled, &t.ChatID, &notify)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				log.Fatal("Task not found")
			}
			log.Fatal(err)
		}
		t.Enabled = enabled == 1
		t.Notify = notify == 1

		fmt.Printf("Running task: %s...\n", t.Name)
		output, err := executor.Run(t)
		if err != nil {
			fmt.Printf("Task failed: %v\n", err)
		} else {
			fmt.Println("Task completed successfully")
		}
		fmt.Println("Output:")
		fmt.Println(output)
	},
}
