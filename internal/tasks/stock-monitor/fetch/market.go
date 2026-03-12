package fetch

import (
	internalhttp "ai-agent/internal/http"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/go-resty/resty/v2"
)

type Quote struct {
	Symbol string
	Price  float64
	Change float64
}

type QuoteProvider string

const (
	QuoteProviderFinnhub QuoteProvider = "finnhub"
	QuoteProviderYahoo   QuoteProvider = "yahoo"

	finnhubToken          = "d6p74epr01qk3chj5sr0d6p74epr01qk3chj5srg"
	finnhubQuoteAPI       = "https://finnhub.io/api/v1/quote"
	finnhubReferer        = "https://finnhub.io/"
	yahooQuoteAPI         = "https://query1.finance.yahoo.com/v7/finance/quote"
	yahooFinanceQuotePage = "https://finance.yahoo.com/quote/%s"
)

type QuoteFetcher interface {
	GetQuote(symbol string) (*Quote, error)
}

type FinnhubQuoteFetcher struct {
	Client *resty.Client
	Token  string
}

type YahooQuoteFetcher struct {
	Client *resty.Client
}

func GetQuote(symbol string) (*Quote, error) {
	return NewQuoteFetcher(QuoteProviderFinnhub).GetQuote(symbol)
}

func GetQuoteByProvider(symbol string, provider QuoteProvider) (*Quote, error) {
	return NewQuoteFetcher(provider).GetQuote(symbol)
}

func NewQuoteFetcher(provider QuoteProvider) QuoteFetcher {
	client := internalhttp.GetClient()

	switch provider {
	case QuoteProviderYahoo:
		return &YahooQuoteFetcher{Client: client}
	case QuoteProviderFinnhub:
		fallthrough
	default:
		return &FinnhubQuoteFetcher{
			Client: client,
			Token:  finnhubToken,
		}
	}
}

func (f *FinnhubQuoteFetcher) GetQuote(symbol string) (*Quote, error) {
	symbol, err := normalizeQuoteSymbol(symbol)
	if err != nil {
		return nil, err
	}

	apiURL := buildFinnhubQuoteURL(symbol, f.Token)
	resp, err := f.client().R().
		SetHeader("Accept", "application/json, text/plain, */*").
		SetHeader("Referer", finnhubReferer).
		SetHeader("Origin", "https://finnhub.io").
		Get(apiURL)
	if err != nil {
		return nil, err
	}

	if resp.IsError() {
		return nil, fmt.Errorf("finnhub unexpected status code: %d", resp.StatusCode())
	}

	var payload struct {
		Current  float64 `json:"c"`
		Change   float64 `json:"dp"`
		Previous float64 `json:"pc"`
	}
	if err := json.Unmarshal(resp.Body(), &payload); err != nil {
		return nil, fmt.Errorf("decode finnhub quote response failed: %w", err)
	}

	if payload.Current == 0 && payload.Previous == 0 {
		return nil, fmt.Errorf("no data")
	}

	return &Quote{
		Symbol: symbol,
		Price:  payload.Current,
		Change: payload.Change,
	}, nil
}

func (f *YahooQuoteFetcher) GetQuote(symbol string) (*Quote, error) {
	symbol, err := normalizeQuoteSymbol(symbol)
	if err != nil {
		return nil, err
	}

	apiURL, referer := buildYahooQuoteRequest(symbol)
	resp, err := f.client().R().
		SetHeader("Accept", "application/json, text/plain, */*").
		SetHeader("Referer", referer).
		SetHeader("Origin", "https://finance.yahoo.com").
		Get(apiURL)
	if err != nil {
		return nil, err
	}

	if resp.IsError() {
		return nil, fmt.Errorf("yahoo unexpected status code: %d", resp.StatusCode())
	}

	var payload struct {
		QuoteResponse struct {
			Result []struct {
				Symbol string  `json:"symbol"`
				Price  float64 `json:"regularMarketPrice"`
				Change float64 `json:"regularMarketChangePercent"`
			} `json:"result"`
		} `json:"quoteResponse"`
	}
	if err := json.Unmarshal(resp.Body(), &payload); err != nil {
		return nil, fmt.Errorf("decode yahoo quote response failed: %w", err)
	}

	if len(payload.QuoteResponse.Result) == 0 {
		return nil, fmt.Errorf("no data")
	}

	item := payload.QuoteResponse.Result[0]
	return &Quote{
		Symbol: item.Symbol,
		Price:  item.Price,
		Change: item.Change,
	}, nil
}

func (f *FinnhubQuoteFetcher) client() *resty.Client {
	if f.Client != nil {
		return f.Client
	}
	return internalhttp.GetClient()
}

func (f *YahooQuoteFetcher) client() *resty.Client {
	if f.Client != nil {
		return f.Client
	}
	return internalhttp.GetClient()
}

func normalizeQuoteSymbol(symbol string) (string, error) {
	symbol = strings.TrimSpace(strings.ToUpper(symbol))
	if symbol == "" {
		return "", fmt.Errorf("symbol is required")
	}
	return symbol, nil
}

func buildFinnhubQuoteURL(symbol, token string) string {
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("token", token)
	return finnhubQuoteAPI + "?" + params.Encode()
}

func buildYahooQuoteRequest(symbol string) (string, string) {
	params := url.Values{}
	params.Set("symbols", symbol)

	apiURL := yahooQuoteAPI + "?" + params.Encode()
	referer := fmt.Sprintf(yahooFinanceQuotePage, url.PathEscape(symbol))
	return apiURL, referer
}
