
# Go CLI Task Manager

Features

- Dynamic tasks
- Bash / Python / Binary execution
- Execution logs
- Telegram notifications
- SQLite (pure Go driver)

## Setup

export TELEGRAM_TOKEN=xxxx
export TELEGRAM_CHAT_ID=xxxx

go mod tidy
go run ./cmd/server


启动服务: go run main.go start
•
列出任务: go run main.go task list
•
添加任务: go run main.go task add --name "Fetch Truth" --type "shell" --command "python3 scripts/fetch_trumpstruth.py" --cron "@every 1h"
•
采集脚本: python3 scripts/fetch_trumpstruth.py (需要安装 requests 库: pip install requests)
