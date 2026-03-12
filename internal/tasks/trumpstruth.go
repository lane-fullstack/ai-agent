package tasks

import (
	"ai-agent/internal/db"
	"ai-agent/internal/executor"
	"ai-agent/internal/http"
	"ai-agent/internal/translator"
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func init() {
	executor.RegisterFunc("FetchTrumpTruths", FetchTrumpTruths)
}

func FetchTrumpTruths(taskId int64) (string, error) {
	database := db.GetDB()
	// The URL to scrape
	url := "https://trumpstruth.org/"

	client := http.GetClient()
	resp, err := client.R().Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch url: %v", err)
	}

	if resp.IsError() {
		return "", fmt.Errorf("received non-200 status code: %d", resp.StatusCode())
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(resp.Body()))
	if err != nil {
		return "", fmt.Errorf("failed to parse html: %v", err)
	}

	// Prepare statements
	checkStmt, err := database.Prepare("SELECT 1 FROM trumpstruth WHERE key = ?")
	if err != nil {
		return "", fmt.Errorf("prepare select failed: %v", err)
	}
	defer checkStmt.Close()

	stmt, err := database.Prepare("INSERT INTO trumpstruth (key, taskname, content) VALUES (?, ?, ?)")
	if err != nil {
		return "", fmt.Errorf("prepare insert failed: %v", err)
	}
	defer stmt.Close()

	count := 0

	// Use a map to track processed IDs in this run
	processedIDs := make(map[string]bool)
	var lastContent []string
	doc.Find(".status").Each(func(i int, s *goquery.Selection) {

		// 获取ID
		url, exists := s.Attr("data-status-url")
		if !exists {
			return
		}

		parts := strings.Split(url, "/statuses/")
		if len(parts) < 2 {
			return
		}

		id := strings.TrimSpace(parts[1])

		if processedIDs[id] {
			return
		}
		processedIDs[id] = true

		// 获取正文
		content := strings.TrimSpace(
			s.Find(".status__content").Text(),
		)

		// 清理文本
		content = strings.Join(strings.Fields(content), " ")

		//if len(content) > 500 {
		//	content = content[:500] + "..."
		//}

		if content == "" {
			return
		}

		// DB check
		var existsDB int
		err = checkStmt.QueryRow(id).Scan(&existsDB)

		if err == nil {
			return
		}

		if !errors.Is(sql.ErrNoRows, err) {
			log.Println("db check error:", err)
			return
		}

		// Translate content
		// We do this AFTER checking DB to save API calls
		translatedContent, err := translator.Translate(content)
		fmt.Println("-----------translatedContent:", translatedContent)
		if err != nil {
			log.Printf("Translation failed for ID %s: %v", id, err)
			// Fallback to original content
		} else {
			lastContent = append(lastContent, translatedContent)
		}

		_, err = stmt.Exec(id, "trumpstruth", content)
		if err != nil {
			log.Println("insert failed:", err)
			return
		}

		count++
	})

	if count == 0 {
		return NoNewContent, nil
	}
	lastContentStr := "川普说:\n"
	if len(lastContent) > 1 {
		lastContentStr = strings.Join(lastContent, "\n---------------------------------\n")
	} else {
		lastContentStr = lastContent[0]
	}

	fmt.Println("-----------lastContentStr:", lastContentStr)
	return lastContentStr, nil
}
