package stock_monitor

import (
	"ai-agent/internal/db"
	"ai-agent/internal/tasks/stock-monitor/fetch"
	"fmt"
	"log"
	"testing"
)

func TestMonitor(t *testing.T) {
	//cfg := config.Load()

	_, err := db.Init("/Users/lane/dev/golang/ai-agent/tasks.db")
	if err != nil {
		log.Fatal(err)
	}
	StockMonitor()
}

func TestGetQuote(t *testing.T) {
	//cfg := config.Load()

	quote, err := fetch.GetQuote("AAPL")
	if err != nil {
		return
	}
	fmt.Println(quote)
}
