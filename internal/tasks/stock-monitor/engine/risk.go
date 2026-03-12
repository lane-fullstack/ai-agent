package engine

import (
	"ai-agent/internal/tasks/stock-monitor/fetch"
	"ai-agent/internal/tasks/stock-monitor/model"
	"fmt"
	"strings"
)

func MatchPortfolio(news fetch.News) []string {

	var hits []string

	title := strings.ToLower(news.Title)

	for _, s := range model.Portfolio {

		if strings.Contains(title, strings.ToLower(s)) {
			hits = append(hits, s)
		}

	}

	return hits
}
func CheckStockMove(symbol string) (bool, float64) {

	q, err := fetch.GetQuote(symbol)
	if err != nil {
		return false, 0
	}

	if q.Change > 4 || q.Change < -4 {
		return true, q.Change
	}

	return false, q.Change
}

func BuildAlert(symbol string, change float64) string {

	return fmt.Sprintf(
		`【美股突发】

和上一轮相比，新增的量化变化：
%s 当前波动 %.2f%%

影响用户账户：
该股票属于核心持仓

相关持仓：
%s

现在最该看什么：
查看是否出现财报 / 做空 / AI行业消息
`,
		symbol,
		change,
		symbol,
	)
}
