
# Go CLI Task Manager

Features

- Dynamic tasks
- Bash / Python / Binary execution
- Execution logs
- Telegram notifications
- SQLite (pure Go driver)

## Setup
* set config.json.dev remove .dev
* get Gemini Api   https://aistudio.google.com/
* get Gmail API https://console.cloud.google.com/apis/api/gmail.googleapis.com

go mod tidy
go run . start


启动服务: go run main.go start
•
列出任务: go run main.go task list
•
添加任务: go run main.go task add --name "Fetch Truth" --type "shell" --command "python3 scripts/fetch_trumpstruth.py" --cron "@every 1h"
•
采集脚本: python3 scripts/fetch_trumpstruth.py (需要安装 requests 库: pip install requests)
