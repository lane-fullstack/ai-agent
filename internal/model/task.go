package model

type Task struct {
	ID      int64
	Name    string
	Type    string
	Command string
	Cron    string
	Enabled bool
	ChatID  int64
	Notify  bool
	Prompt  string
}
