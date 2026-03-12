package provider

//
//import (
//	"bufio"
//	"context"
//	"crypto/rand"
//	"crypto/sha256"
//	"encoding/base64"
//	"encoding/json"
//	"errors"
//	"fmt"
//	"io"
//	"net"
//	"net/http"
//	"net/url"
//	"os"
//	"os/exec"
//	"path/filepath"
//	"runtime"
//	"strings"
//	"sync"
//	"time"
//)
//
//const (
//	// 你需要替换成可用的 OAuth Client ID
//	// 注意：不是随便一个 OpenAI App Client ID 都能用
//	clientID = "pS769Y3m9Iis83Y1kS77JAszE9uDk07v"
//
//	authURLBase  = "https://auth.openai.com/oauth/authorize"
//	tokenURLBase = "https://auth.openai.com/oauth/token"
//
//	callbackHost = "127.0.0.1"
//	callbackPort = 1455
//	callbackPath = "/auth/callback"
//)
//
//var (
//	//redirectURL = fmt.Sprintf("http://%s:%d%s", callbackHost, callbackPort, callbackPath)
//	redirectURL = "com.openai.provider://auth0.openai.com/ios/com.openai.provider/callback"
//	scopes      = []string{"openid", "profile", "email", "offline_access"}
//)
//
//type OAuthTokenResponse struct {
//	AccessToken  string `json:"access_token"`
//	RefreshToken string `json:"refresh_token"`
//	IDToken      string `json:"id_token"`
//	TokenType    string `json:"token_type"`
//	ExpiresIn    int64  `json:"expires_in"`
//	Scope        string `json:"scope"`
//}
//
//type StoredAuthProfile struct {
//	Provider   string `json:"provider"`
//	Access     string `json:"access"`
//	Refresh    string `json:"refresh"`
//	Expires    int64  `json:"expires"`
//	AccountID  string `json:"accountId,omitempty"`
//	TokenType  string `json:"tokenType,omitempty"`
//	Scope      string `json:"scope,omitempty"`
//	ReceivedAt int64  `json:"receivedAt,omitempty"`
//}
//
//type authResult struct {
//	Code  string
//	State string
//	Err   error
//}
//
//// -------------------- PKCE --------------------
//
//func generateRandomURLSafeString(rawLen int) (string, error) {
//	b := make([]byte, rawLen)
//	if _, err := rand.Read(b); err != nil {
//		return "", err
//	}
//	return base64.RawURLEncoding.EncodeToString(b), nil
//}
//
//func generateCodeChallenge(verifier string) string {
//	sum := sha256.Sum256([]byte(verifier))
//	return base64.RawURLEncoding.EncodeToString(sum[:])
//}
//
//// -------------------- Browser --------------------
//
//func openBrowser(u string) error {
//	switch runtime.GOOS {
//	case "darwin":
//		return exec.Command("open", u).Start()
//	case "linux":
//		return exec.Command("xdg-open", u).Start()
//	case "windows":
//		return exec.Command("rundll32", "url.dll,FileProtocolHandler", u).Start()
//	default:
//		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
//	}
//}
//
//// -------------------- Callback Server --------------------
//
//func startCallbackServer(expectedState string) (func(context.Context) error, <-chan authResult, error) {
//	mux := http.NewServeMux()
//	resultCh := make(chan authResult, 1)
//
//	server := &http.Server{
//		Addr:              fmt.Sprintf("%s:%d", callbackHost, callbackPort),
//		Handler:           mux,
//		ReadHeaderTimeout: 5 * time.Second,
//	}
//
//	var once sync.Once
//	send := func(res authResult) {
//		once.Do(func() {
//			resultCh <- res
//		})
//	}
//
//	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
//		q := r.URL.Query()
//		if errStr := q.Get("error"); errStr != "" {
//			desc := q.Get("error_description")
//			http.Error(w, "OAuth error: "+errStr+" "+desc, http.StatusBadRequest)
//			send(authResult{Err: fmt.Errorf("oauth error: %s %s", errStr, desc)})
//			return
//		}
//
//		code := q.Get("code")
//		state := q.Get("state")
//		if code == "" {
//			http.Error(w, "missing code", http.StatusBadRequest)
//			send(authResult{Err: errors.New("missing code in callback")})
//			return
//		}
//		if state == "" {
//			http.Error(w, "missing state", http.StatusBadRequest)
//			send(authResult{Err: errors.New("missing state in callback")})
//			return
//		}
//		if state != expectedState {
//			http.Error(w, "invalid state", http.StatusBadRequest)
//			send(authResult{Err: errors.New("state mismatch")})
//			return
//		}
//
//		w.Header().Set("Content-Type", "text/html; charset=utf-8")
//		_, _ = io.WriteString(w, `
//<!doctype html>
//<html>
//<head><meta charset="utf-8"><title>Login success</title></head>
//<body>
//  <h2>Login success</h2>
//  <p>You can close this window and return to the terminal.</p>
//</body>
//</html>`)
//
//		send(authResult{Code: code, State: state})
//	})
//
//	ln, err := net.Listen("tcp", server.Addr)
//	if err != nil {
//		return nil, nil, err
//	}
//
//	go func() {
//		if err := server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
//			send(authResult{Err: err})
//		}
//	}()
//
//	return server.Shutdown, resultCh, nil
//}
//
//// -------------------- Manual Paste Fallback --------------------
//
//func readManualCode(expectedState string) (string, error) {
//	fmt.Println("\n如果浏览器没有自动回调，请粘贴以下任一内容：")
//	fmt.Println("1) 完整回调 URL")
//	fmt.Println("2) 仅 code")
//	fmt.Println("3) code#state")
//	fmt.Print("\nPaste here: ")
//
//	reader := bufio.NewReader(os.Stdin)
//	text, err := reader.ReadString('\n')
//	if err != nil {
//		return "", err
//	}
//	text = strings.TrimSpace(text)
//	if text == "" {
//		return "", errors.New("empty input")
//	}
//
//	// 1) 完整 URL
//	if strings.HasPrefix(text, "http://") || strings.HasPrefix(text, "https://") {
//		u, err := url.Parse(text)
//		if err != nil {
//			return "", err
//		}
//		code := u.Query().Get("code")
//		state := u.Query().Get("state")
//		if code == "" {
//			return "", errors.New("missing code in pasted URL")
//		}
//		if state != "" && state != expectedState {
//			return "", errors.New("state mismatch in pasted URL")
//		}
//		return code, nil
//	}
//
//	// 2) code#state
//	if strings.Contains(text, "#") {
//		parts := strings.SplitN(text, "#", 2)
//		code := strings.TrimSpace(parts[0])
//		state := strings.TrimSpace(parts[1])
//		if code == "" {
//			return "", errors.New("missing code")
//		}
//		if state != "" && state != expectedState {
//			return "", errors.New("state mismatch")
//		}
//		return code, nil
//	}
//
//	// 3) 仅 code
//	return text, nil
//}
//
//// -------------------- Token Exchange --------------------
//
//func exchangeToken(ctx context.Context, code, verifier string) (*OAuthTokenResponse, error) {
//	form := url.Values{}
//	form.Set("grant_type", "authorization_code")
//	form.Set("client_id", clientID)
//	form.Set("code", code)
//	form.Set("code_verifier", verifier)
//	form.Set("redirect_uri", redirectURL)
//
//	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURLBase, strings.NewReader(form.Encode()))
//	if err != nil {
//		return nil, err
//	}
//	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
//
//	client := &http.Client{Timeout: 20 * time.Second}
//	resp, err := client.Do(req)
//	if err != nil {
//		return nil, err
//	}
//	defer resp.Body.Close()
//
//	body, _ := io.ReadAll(resp.Body)
//	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
//		return nil, fmt.Errorf("token exchange failed: status=%d body=%s", resp.StatusCode, string(body))
//	}
//
//	var tr OAuthTokenResponse
//	if err := json.Unmarshal(body, &tr); err != nil {
//		return nil, fmt.Errorf("decode token response: %w, body=%s", err, string(body))
//	}
//	if tr.AccessToken == "" {
//		return nil, fmt.Errorf("empty access_token in response: %s", string(body))
//	}
//	return &tr, nil
//}
//
//// -------------------- JWT Parse --------------------
//
//func parseJWTClaims(jwt string) (map[string]any, error) {
//	parts := strings.Split(jwt, ".")
//	if len(parts) < 2 {
//		return nil, errors.New("invalid jwt format")
//	}
//
//	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
//	if err != nil {
//		return nil, err
//	}
//
//	var claims map[string]any
//	if err := json.Unmarshal(payload, &claims); err != nil {
//		return nil, err
//	}
//	return claims, nil
//}
//
//func extractAccountID(accessToken string) string {
//	claims, err := parseJWTClaims(accessToken)
//	if err != nil {
//		return ""
//	}
//
//	// 尝试多个常见字段
//	candidates := []string{
//		"https://api.openai.com/auth/chatgpt_account_id",
//		"chatgpt_account_id",
//		"account_id",
//		"sub",
//	}
//	for _, k := range candidates {
//		if v, ok := claims[k]; ok {
//			if s, ok := v.(string); ok && s != "" {
//				return s
//			}
//		}
//	}
//	return ""
//}
//
//// -------------------- Storage --------------------
//
//func defaultAuthPath() (string, error) {
//	home, err := os.UserHomeDir()
//	if err != nil {
//		return "", err
//	}
//	dir := filepath.Join(home, ".mycodex")
//	if err := os.MkdirAll(dir, 0o700); err != nil {
//		return "", err
//	}
//	return filepath.Join(dir, "auth.json"), nil
//}
//
//func saveAuthProfile(path string, p *StoredAuthProfile) error {
//	data, err := json.MarshalIndent(p, "", "  ")
//	if err != nil {
//		return err
//	}
//	return os.WriteFile(path, data, 0o600)
//}
//
//// -------------------- Main Flow --------------------
//
//func buildAuthorizeURL(state, verifier string) string {
//	challenge := generateCodeChallenge(verifier)
//
//	u, _ := url.Parse(authURLBase)
//	q := u.Query()
//	q.Set("client_id", clientID)
//	q.Set("redirect_uri", redirectURL)
//	q.Set("response_type", "code")
//	q.Set("scope", strings.Join(scopes, " "))
//	q.Set("state", state)
//	q.Set("code_challenge", challenge)
//	q.Set("code_challenge_method", "S256")
//	u.RawQuery = q.Encode()
//	return u.String()
//}
//
//func login() error {
//
//	verifier, err := generateRandomURLSafeString(48)
//	if err != nil {
//		return err
//	}
//	state, err := generateRandomURLSafeString(24)
//	if err != nil {
//		return err
//	}
//
//	authURL := buildAuthorizeURL(state, verifier)
//
//	shutdown, callbackCh, err := startCallbackServer(state)
//	if err != nil {
//		return fmt.Errorf("start callback server: %w", err)
//	}
//	defer func() {
//		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
//		defer cancel()
//		_ = shutdown(ctx)
//	}()
//
//	fmt.Println("OpenAI Codex OAuth login")
//	fmt.Println("Callback:", redirectURL)
//	fmt.Println("Opening browser...")
//	fmt.Println(authURL)
//	fmt.Println()
//
//	if err := openBrowser(authURL); err != nil {
//		fmt.Printf("failed to open browser automatically: %v\n", err)
//		fmt.Println("Please open the URL manually.")
//	}
//
//	var code string
//
//	select {
//	case res := <-callbackCh:
//		if res.Err != nil {
//			return fmt.Errorf("callback failed: %w", res.Err)
//		}
//		code = res.Code
//		fmt.Println("Callback received.")
//	case <-time.After(90 * time.Second):
//		fmt.Println("No callback received within 90s.")
//		manualCode, err := readManualCode(state)
//		if err != nil {
//			return fmt.Errorf("manual input failed: %w", err)
//		}
//		code = manualCode
//	}
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//
//	tokenResp, err := exchangeToken(ctx, code, verifier)
//	if err != nil {
//		return err
//	}
//
//	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second).Unix()
//	accountID := extractAccountID(tokenResp.AccessToken)
//
//	profile := &StoredAuthProfile{
//		Provider:   "openai-codex",
//		Access:     tokenResp.AccessToken,
//		Refresh:    tokenResp.RefreshToken,
//		Expires:    expiresAt,
//		AccountID:  accountID,
//		TokenType:  tokenResp.TokenType,
//		Scope:      tokenResp.Scope,
//		ReceivedAt: time.Now().Unix(),
//	}
//
//	path, err := defaultAuthPath()
//	if err != nil {
//		return err
//	}
//	if err := saveAuthProfile(path, profile); err != nil {
//		return err
//	}
//
//	fmt.Println("\nLogin success.")
//	fmt.Printf("Saved to: %s\n", path)
//	fmt.Printf("AccountID: %s\n", profile.AccountID)
//	fmt.Printf("Expires : %s\n", time.Unix(profile.Expires, 0).Format(time.RFC3339))
//
//	if len(profile.Access) > 20 {
//		fmt.Printf("Access  : %s...\n", profile.Access[:20])
//	}
//	if len(profile.Refresh) > 20 {
//		fmt.Printf("Refresh : %s...\n", profile.Refresh[:20])
//	}
//
//	return nil
//}
//
//func main() {
//	if err := login(); err != nil {
//		fmt.Fprintln(os.Stderr, "login failed:", err)
//		os.Exit(1)
//	}
//}
