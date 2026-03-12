package stock_monitor

import (
	"ai-agent/internal/tasks/stock-monitor/engine"
	"ai-agent/internal/tasks/stock-monitor/fetch"
	"ai-agent/internal/tasks/stock-monitor/model"
	"fmt"
)

func StockMonitor() {

	// 1 抓新闻
	news, err := fetch.FetchYahooNews()
	//fmt.Println("news:", news)
	if err != nil {
		panic(err)
	}

	for _, n := range news {

		hits := engine.MatchPortfolio(n)
		fmt.Println("hits:", hits)
		if len(hits) > 0 {

			fmt.Println("相关新闻:", n.Title)

		}

	}

	// 2 检查股票波动

	for _, s := range model.Portfolio {

		ok, change := engine.CheckStockMove(s)

		if ok {

			alert := engine.BuildAlert(s, change)

			fmt.Println(alert)

		}

	}
}
