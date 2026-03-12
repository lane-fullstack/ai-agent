package scheduler

import (
	"ai-agent/internal/executor"
	"ai-agent/internal/model"
	"ai-agent/internal/provider"
	"ai-agent/internal/tasks"
	"database/sql"
	"log"
	"time"

	"github.com/robfig/cron/v3"
)

type Notifier interface {
	Send(chatID int64, text string)
}

type Scheduler struct {
	cron     *cron.Cron
	db       *sql.DB
	notifier Notifier
}

func New(db *sql.DB, notifier Notifier) *Scheduler {

	return &Scheduler{
		cron:     cron.New(),
		db:       db,
		notifier: notifier,
	}
}

func (s *Scheduler) Start() {
	s.cron.Start()
}

func (s *Scheduler) LoadTasks(l provider.LLM) error {

	rows, err := s.db.Query("SELECT id,name,type,command,cron,enabled,chat_id,notify,prompt FROM tasks")
	if err != nil {
		return err
	}

	for rows.Next() {

		var t model.Task
		var enabled int
		var notify int

		err = rows.Scan(&t.ID, &t.Name, &t.Type, &t.Command, &t.Cron, &enabled, &t.ChatID, &notify, &t.Prompt)
		if err != nil {
			log.Println("Error scanning task:", err)
			continue
		}

		t.Enabled = enabled == 1
		t.Notify = notify == 1

		if t.Enabled {

			task := t

			s.cron.AddFunc(task.Cron, func() {
				s.runTask(task)
			})

			if t.Prompt != "" {
				l.SetTaskPrompt(task.ID, t.Prompt)
			}

		}
	}

	return nil
}

func (s *Scheduler) runTask(task model.Task) {
	start := time.Now()
	output, err := executor.Run(task)

	if output == tasks.NoNewContent {
		status := "no_content"
		s.db.Exec(
			"INSERT INTO task_runs(task_id,start_time,end_time,status,output) VALUES(?,?,?,?,?)",
			task.ID, start.String(), time.Now().String(), status, "No new content",
		)
		log.Printf("Task %s: No new content", task.Name)
		return
	}

	end := time.Now()
	status := "success"
	if err != nil {
		status = "failed"
	}

	s.db.Exec(
		"INSERT INTO task_runs(task_id,start_time,end_time,status,output) VALUES(?,?,?,?,?)",
		task.ID, start.String(), end.String(), status, output,
	)

	if task.Prompt != "" {
		output, err = provider.L.GenerateOneShot(task.ID, task.Prompt)
		if err != nil {
			log.Println("Error generating one-shot response:", err)
		}
	}

	if task.Notify {
		msg := "Task: " + task.Name + "\nStatus: " + status + "\nOutput:\n" + output
		s.notifier.Send(task.ChatID, msg)
	}
}
