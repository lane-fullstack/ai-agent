#!/bin/zsh
set -euo pipefail
export PATH="/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"
export GOG_ACCOUNT="ruok6688@gmail.com"

if ! command -v gog >/dev/null 2>&1; then
  echo "gog not found" >&2
  exit 1
fi

# 只保留个人和学校邮件；其余包括银行/平台/通知超过3天全部删（排除 Trash）
QUERY='older_than:3d -in:trash -from:(ruok6688@gmail.com OR lei20221012@gmail.com OR @gmail.com OR @icloud.com OR @me.com OR @qq.com OR @outlook.com OR @hotmail.com OR @yahoo.com OR @ausd.us OR @teachers.ausd.us OR @sgusd.k12.ca.us OR @wukongsch.com)'

gog gmail trash --account "$GOG_ACCOUNT" --query "$QUERY" --max 1000 --force --json
