package translator

import (
	"ai-agent/internal/config"
	"ai-agent/internal/http"
	"encoding/json"
	"fmt"
)

type TranslateRequest struct {
	Texts  []string `json:"texts"`
	Source string   `json:"source"`
	Target string   `json:"target"`
}

type TranslateResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Source       string   `json:"source"`
		Target       string   `json:"target"`
		Translations []string `json:"translations"`
	} `json:"data"`
}

func Translate(text string) (string, error) {
	cfg := config.Load()
	url := config.AsString(cfg["TranslateAPI"])
	apiKey := config.AsString(cfg["TranslateKey"])

	reqBody := TranslateRequest{
		Texts:  []string{text},
		Source: "en",
		Target: "zh",
	}

	client := http.GetClient()
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("X-API-KEY", apiKey).
		SetBody(reqBody).
		Post(url)

	if err != nil {
		return "", fmt.Errorf("translation request failed: %v", err)
	}

	if resp.IsError() {
		return "", fmt.Errorf("translation API error: %d %s", resp.StatusCode(), resp.String())
	}

	var result TranslateResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return "", fmt.Errorf("failed to decode translation response: %v", err)
	}

	if result.Code != 0 {
		return "", fmt.Errorf("translation API returned error: %s", result.Msg)
	}

	if len(result.Data.Translations) == 0 {
		return "", fmt.Errorf("no translations returned")
	}

	return result.Data.Translations[0], nil
}
