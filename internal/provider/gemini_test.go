package provider

import (
	"fmt"
	"testing"
)

func TestChat(t *testing.T) {
	provider, err := NewGeminiProvider("AIzaSyC6vchUjxGIF0ynvvrL-BDfQPrG6MaUQck")
	if err != nil {
		return
	}
	shot, err := provider.GenerateOneShot(1, "你好")
	if err != nil {
		return
	}
	fmt.Println(shot)
}
