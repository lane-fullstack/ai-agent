package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
)

const (
	credentialsFile = "./data/credentials/google_ruok6688.json"
	tokenFile       = "./data/credentials/token.json"
)

func CleanMail(taskId int64) {

	ctx := context.Background()

	b, err := os.ReadFile(credentialsFile)
	if err != nil {
		log.Fatal("Unable to read credentials:", err)
	}

	config, err := google.ConfigFromJSON(
		b,
		gmail.GmailModifyScope,
	)

	if err != nil {
		log.Fatal(err)
	}

	config.RedirectURL = "http://localhost:8080"

	tokenChan := make(chan *oauth2.Token)

	// HTTP callback server
	go func() {

		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

			code := r.URL.Query().Get("code")

			if code == "" {
				http.Error(w, "Missing code", 400)
				return
			}

			tok, err := config.Exchange(ctx, code)
			if err != nil {
				http.Error(w, "Token exchange failed", 500)
				log.Fatal(err)
			}

			saveToken(tokenFile, tok)

			fmt.Fprintf(w, "Login successful. You can close this window.")

			tokenChan <- tok
		})

		fmt.Println("Listening on http://localhost:8080")
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)

	fmt.Println("Open this URL in browser:")
	fmt.Println(authURL)

	var tok *oauth2.Token

	select {
	case tok = <-tokenChan:
	case <-time.After(120 * time.Second):
		log.Fatal("OAuth timeout")
	}

	client := config.Client(ctx, tok)

	srv, err := gmail.New(client)
	if err != nil {
		log.Fatal(err)
	}

	// 示例：查询邮件
	res, err := srv.Users.Messages.List("me").
		Q("older_than:3d -in:trash").
		MaxResults(10).
		Do()

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Found messages:", len(res.Messages))
}

func saveToken(path string, token *oauth2.Token) {

	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	json.NewEncoder(f).Encode(token)
}
