package http

import (
	"time"

	"github.com/go-resty/resty/v2"
)

var Client *resty.Client

func Init() {
	if Client == nil {
		Client = resty.New()
		Client.SetTimeout(1 * time.Minute)
		Client.SetHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	}
}

func GetClient() *resty.Client {
	if Client == nil {
		Init()
	}
	return Client
}
