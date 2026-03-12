package fetch

import (
	"ai-agent/internal/db"
	"ai-agent/internal/http"
	"bytes"
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const yahooFinanceBaseURL = "https://finance.yahoo.com"

type News struct {
	Title string
	Link  string
}

func FetchYahooNews() ([]News, error) {
	client := http.GetClient()
	resp, err := client.R().Get(yahooFinanceBaseURL + "/news/")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch yahoo news: %w", err)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("received non-200 status code: %d", resp.StatusCode())
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(resp.Body()))
	if err != nil {
		return nil, fmt.Errorf("failed to parse html: %w", err)
	}

	candidates := collectYahooNews(doc)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no news found")
	}

	news, err := persistNewYahooNews(candidates)
	if err != nil {
		return nil, err
	}

	return news, nil
}

func collectYahooNews(doc *goquery.Document) []News {
	list := make([]News, 0)
	seen := make(map[string]struct{})

	appendNews := func(title, link string) {
		title = strings.TrimSpace(title)
		link = normalizeYahooNewsLink(link)
		if title == "" || link == "" || isIgnoredYahooNewsLink(link) {
			return
		}

		if _, ok := seen[link]; ok {
			return
		}

		seen[link] = struct{}{}
		list = append(list, News{
			Title: title,
			Link:  link,
		})
	}

	doc.Find("a h3").Each(func(i int, s *goquery.Selection) {
		parent := s.Closest("a")
		if parent.Length() == 0 {
			return
		}

		title := strings.TrimSpace(s.Text())
		if title == "" {
			title, _ = parent.Attr("title")
		}
		if title == "" {
			title, _ = parent.Attr("aria-label")
		}

		link, _ := parent.Attr("href")
		appendNews(title, link)
	})

	doc.Find("h3 a").Each(func(i int, s *goquery.Selection) {
		title := strings.TrimSpace(s.Text())
		link, _ := s.Attr("href")
		appendNews(title, link)
	})

	return list
}

func normalizeYahooNewsLink(link string) string {
	link = strings.TrimSpace(link)
	if link == "" {
		return ""
	}

	if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
		return link
	}

	if strings.HasPrefix(link, "/") {
		return yahooFinanceBaseURL + link
	}

	return yahooFinanceBaseURL + "/" + link
}

func isIgnoredYahooNewsLink(link string) bool {
	if !isYahooNewsPath(link) {
		return true
	}

	return strings.Contains(link, "my.yahoo.com") ||
		strings.Contains(link, "login.yahoo.com") ||
		strings.Contains(link, "/m/") ||
		strings.Contains(link, "/ad/")
}

func isYahooNewsPath(link string) bool {
	return strings.HasPrefix(link, yahooFinanceBaseURL+"/news/") ||
		strings.HasPrefix(link, yahooFinanceBaseURL+"/news")
}

func persistNewYahooNews(candidates []News) ([]News, error) {
	database := db.GetDB()
	keys := make([]string, 0, len(candidates))
	for _, item := range candidates {
		keys = append(keys, item.Link)
	}

	existing, err := loadExistingTrumpstruthKeys(database, keys)
	if err != nil {
		return nil, fmt.Errorf("load existing yahoo news failed: %w", err)
	}

	news := make([]News, 0, len(candidates))
	for _, item := range candidates {
		if _, ok := existing[item.Link]; ok {
			continue
		}
		news = append(news, item)
	}

	if len(news) == 0 {
		return news, nil
	}

	if err := insertYahooNewsBatch(database, news); err != nil {
		return nil, fmt.Errorf("insert yahoo news failed: %w", err)
	}

	return news, nil
}

func loadExistingTrumpstruthKeys(database *sql.DB, keys []string) (map[string]struct{}, error) {
	existing := make(map[string]struct{})
	if len(keys) == 0 {
		return existing, nil
	}

	placeholders := make([]string, 0, len(keys))
	args := make([]any, 0, len(keys))
	for _, key := range keys {
		placeholders = append(placeholders, "?")
		args = append(args, key)
	}

	query := "SELECT key FROM trumpstruth WHERE key IN (" + strings.Join(placeholders, ",") + ")"
	rows, err := database.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		existing[key] = struct{}{}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return existing, nil
}

func insertYahooNewsBatch(database *sql.DB, news []News) error {
	tx, err := database.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare("INSERT INTO trumpstruth (key, taskname, content) VALUES (?, ?, ?)")
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, item := range news {
		if _, err := stmt.Exec(item.Link, "stock-monitor-news", item.Title); err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	log.Printf("inserted %d new yahoo news rows into trumpstruth", len(news))
	return nil
}
