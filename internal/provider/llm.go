package provider

import (
	"context"
	"time"
)

var L LLM

type LLM interface {
	ListModelNames(ctx context.Context) ([]string, error)
	SetHistoryDir(dir string) error
	SetMaxHistory(n int)
	SetTimeout(d time.Duration)
	SetPreferredModels(models []string)
	CurrentModel() string
	SetTaskPrompt(taskID int64, prompt string) error
	ClearTask(taskID int64) error
	GenerateOneShot(taskID int64, prompt string) (string, error)
	Chat(taskID int64, prompt string) (string, error)
}
