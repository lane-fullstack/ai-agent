package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"google.golang.org/genai"
)

const (
	DefaultHistoryDir = ".history"
	DefaultMaxHistory = 20 // 20 条消息，不是 20 轮
	DefaultTimeout    = 90 * time.Second
)

// HistoryContent 用于本地 JSON 持久化
type HistoryContent struct {
	Role  string   `json:"role"`
	Parts []string `json:"parts"`
}

// TaskContext 每个 task 独立上下文
type TaskContext struct {
	SystemPrompt string           `json:"system_prompt,omitempty"`
	History      []*genai.Content `json:"-"`
	mu           sync.Mutex       `json:"-"`
}

// GeminiProvider 线程安全
type GeminiProvider struct {
	client *genai.Client

	// 当前主模型；失败时可轮换
	model string

	// 优先模型列表，会在初始化时和真实可用模型做交集
	modelCandidates []string

	// task 级上下文
	tasks sync.Map // map[int64]*TaskContext

	// 配置
	historyDir string
	maxHistory int
	timeout    time.Duration
}

// NewGeminiProvider 创建 provider
func NewGeminiProvider(apiKey string) (*GeminiProvider, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("empty Gemini API key")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("create genai client: %w", err)
	}

	p := &GeminiProvider{
		client: client,
		// 这里放“偏好顺序”，后面会自动和真实可用模型做交集
		modelCandidates: []string{
			"gemini-3-flash-preview",
			"gemini-2.0-flash",
			"gemini-2.0-flash-lite",
			"gemini-1.5-pro",
		},
		historyDir: DefaultHistoryDir,
		maxHistory: DefaultMaxHistory,
		timeout:    DefaultTimeout,
	}

	if err = os.MkdirAll(p.historyDir, 0o755); err != nil {
		return nil, fmt.Errorf("create history dir: %w", err)
	}

	// 自动探测可用模型；失败就保留默认候选
	if err = p.initModel(ctx); err != nil {
		log.Printf("GeminiProvider: initModel failed, fallback to static candidates: %v", err)
		if len(p.modelCandidates) == 0 {
			return nil, errors.New("no model candidates available")
		}
		p.model = p.modelCandidates[0]
	}

	return p, nil
}

// initModel 探测当前账号实际可用模型
func (p *GeminiProvider) initModel(ctx context.Context) error {
	available, err := p.ListModelNames(ctx)
	if err != nil {
		return err
	}
	if len(available) == 0 {
		return errors.New("no models returned by API")
	}

	availSet := make(map[string]struct{}, len(available))
	for _, m := range available {
		availSet[m] = struct{}{}
	}

	filtered := make([]string, 0, len(p.modelCandidates))
	for _, candidate := range p.modelCandidates {
		if _, ok := availSet[candidate]; ok {
			filtered = append(filtered, candidate)
		}
	}

	// 如果偏好列表一个都没命中，就退化为“所有 generateContent 能用的模型”
	if len(filtered) == 0 {
		filtered = append(filtered, available...)
	}

	p.modelCandidates = filtered
	p.model = filtered[0]
	return nil
}

// ListModelNames 返回当前可用模型列表（只保留名称）
func (p *GeminiProvider) ListModelNames(ctx context.Context) ([]string, error) {

	page, err := p.client.Models.List(ctx, nil)
	if err != nil {
		return nil, err
	}

	var names []string

	for {

		for _, model := range page.Items {

			if model == nil {
				continue
			}

			name := strings.TrimPrefix(model.Name, "models/")
			names = append(names, name)
		}

		if page.NextPageToken == "" {
			break
		}

		page, err = p.client.Models.List(ctx, &genai.ListModelsConfig{
			PageToken: page.NextPageToken,
		})

		if err != nil {
			return nil, err
		}
	}

	return names, nil
}

// SetHistoryDir 修改历史目录
func (p *GeminiProvider) SetHistoryDir(dir string) error {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return errors.New("empty history dir")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	p.historyDir = dir
	return nil
}

// SetMaxHistory 修改保留历史条数
func (p *GeminiProvider) SetMaxHistory(n int) {
	if n > 0 {
		p.maxHistory = n
	}
}

// SetTimeout 修改请求超时
func (p *GeminiProvider) SetTimeout(d time.Duration) {
	if d > 0 {
		p.timeout = d
	}
}

// SetPreferredModels 自定义优先模型列表
func (p *GeminiProvider) SetPreferredModels(models []string) {
	var out []string
	for _, m := range models {
		m = strings.TrimSpace(m)
		if m != "" {
			out = append(out, m)
		}
	}
	if len(out) > 0 {
		p.modelCandidates = uniqueStrings(out)
		p.model = p.modelCandidates[0]
	}
}

// CurrentModel 当前生效模型
func (p *GeminiProvider) CurrentModel() string {
	return p.model
}

// SetTaskPrompt 设置某个 task 的 system prompt
func (p *GeminiProvider) SetTaskPrompt(taskID int64, prompt string) error {
	tc, err := p.getOrCreateTask(taskID)
	if err != nil {
		return err
	}

	tc.mu.Lock()
	defer tc.mu.Unlock()

	tc.SystemPrompt = strings.TrimSpace(prompt)

	// 尝试加载已有历史
	if len(tc.History) == 0 {
		if history, err := p.loadHistory(taskID); err == nil {
			tc.History = history
		}
	}

	return nil
}

// ClearTask 清空某个 task 的内存和本地历史
func (p *GeminiProvider) ClearTask(taskID int64) error {
	p.tasks.Delete(taskID)
	filename := p.getHistoryFileName(taskID)
	if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Chat 多轮对话
func (p *GeminiProvider) Chat(taskID int64, prompt string) (string, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return "", errors.New("empty prompt")
	}

	tc, err := p.getOrCreateTask(taskID)
	if err != nil {
		return "", err
	}

	tc.mu.Lock()
	defer tc.mu.Unlock()

	// 懒加载历史
	if len(tc.History) == 0 {
		if history, err := p.loadHistory(taskID); err == nil {
			tc.History = history
		}
	}

	// 添加用户输入
	tc.History = append(tc.History, userText(prompt))
	tc.History = pruneHistory(tc.History, p.maxHistory)

	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()

	resp, err := p.generateWithRetry(ctx, tc.SystemPrompt, tc.History)
	if err != nil {
		return "", err
	}

	text := strings.TrimSpace(resp.Text())
	if text == "" {
		text = "(empty response)"
	}

	tc.History = append(tc.History, modelText(text))
	tc.History = pruneHistory(tc.History, p.maxHistory)

	if err := p.saveHistory(taskID, tc.History); err != nil {
		log.Printf("save history failed, task=%d err=%v", taskID, err)
	}

	return text, nil
}

// GenerateOneShot 单次生成，不读写历史
func (p *GeminiProvider) GenerateOneShot(taskID int64, prompt string) (string, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return "", errors.New("empty prompt")
	}

	tc, err := p.getOrCreateTask(taskID)
	if err != nil {
		return "", err
	}

	tc.mu.Lock()
	systemPrompt := tc.SystemPrompt
	tc.mu.Unlock()

	contents := []*genai.Content{
		userText(prompt),
	}

	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()

	resp, err := p.generateWithRetry(ctx, systemPrompt, contents)
	if err != nil {
		return "", err
	}

	text := strings.TrimSpace(resp.Text())
	if text == "" {
		text = "(empty response)"
	}
	return text, nil
}

// generateWithRetry 轮换模型重试
func (p *GeminiProvider) generateWithRetry(
	ctx context.Context,
	systemPrompt string,
	contents []*genai.Content,
) (*genai.GenerateContentResponse, error) {
	if len(contents) == 0 {
		return nil, errors.New("empty contents")
	}
	if len(p.modelCandidates) == 0 {
		return nil, errors.New("no model candidates configured")
	}

	var lastErr error

	for i := 0; i < len(p.modelCandidates); i++ {
		modelName := p.modelCandidates[i]

		cfg := &genai.GenerateContentConfig{}
		if strings.TrimSpace(systemPrompt) != "" {
			cfg.SystemInstruction = &genai.Content{
				Parts: []*genai.Part{
					{Text: systemPrompt},
				},
			}
		}

		resp, err := p.client.Models.GenerateContent(ctx, modelName, contents, cfg)
		if err == nil {
			p.model = modelName
			return resp, nil
		}

		lastErr = err
		errText := strings.ToLower(err.Error())

		// 可重试：限流 / quota / 暂时资源不足 / 模型不存在
		if strings.Contains(errText, "429") ||
			strings.Contains(errText, "quota") ||
			strings.Contains(errText, "resource exhausted") ||
			strings.Contains(errText, "not found") ||
			strings.Contains(errText, "unsupported") {
			log.Printf("Gemini model %s failed, try next. err=%v", modelName, err)
			continue
		}

		// 其它错误直接返回
		return nil, err
	}

	return nil, fmt.Errorf("all candidate models failed: %w", lastErr)
}

// getOrCreateTask 获取 task context
func (p *GeminiProvider) getOrCreateTask(taskID int64) (*TaskContext, error) {
	if taskID == 0 {
		return nil, errors.New("taskID cannot be 0")
	}

	if v, ok := p.tasks.Load(taskID); ok {
		return v.(*TaskContext), nil
	}

	tc := &TaskContext{}
	actual, _ := p.tasks.LoadOrStore(taskID, tc)
	return actual.(*TaskContext), nil
}

// pruneHistory 只保留最近 max 条
func pruneHistory(history []*genai.Content, max int) []*genai.Content {
	if max <= 0 || len(history) <= max {
		return history
	}
	return slices.Clone(history[len(history)-max:])
}

// saveHistory 保存历史
func (p *GeminiProvider) saveHistory(taskID int64, history []*genai.Content) error {
	filename := p.getHistoryFileName(taskID)

	list := make([]HistoryContent, 0, len(history))
	for _, c := range history {
		if c == nil {
			continue
		}
		var parts []string
		for _, part := range c.Parts {
			if part == nil {
				continue
			}
			parts = append(parts, part.Text)
		}
		list = append(list, HistoryContent{
			Role:  c.Role,
			Parts: parts,
		})
	}

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal history: %w", err)
	}

	return os.WriteFile(filename, data, 0o644)
}

// loadHistory 读取历史
func (p *GeminiProvider) loadHistory(taskID int64) ([]*genai.Content, error) {
	filename := p.getHistoryFileName(taskID)

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var list []HistoryContent
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("unmarshal history: %w", err)
	}

	out := make([]*genai.Content, 0, len(list))
	for _, item := range list {
		parts := make([]*genai.Part, 0, len(item.Parts))
		for _, text := range item.Parts {
			parts = append(parts, &genai.Part{Text: text})
		}
		out = append(out, &genai.Content{
			Role:  item.Role,
			Parts: parts,
		})
	}
	return out, nil
}

func (p *GeminiProvider) getHistoryFileName(taskID int64) string {
	return filepath.Join(p.historyDir, fmt.Sprintf("%d.json", taskID))
}

func userText(text string) *genai.Content {
	return &genai.Content{
		Role: "user",
		Parts: []*genai.Part{
			{Text: text},
		},
	}
}

func modelText(text string) *genai.Content {
	return &genai.Content{
		Role: "model",
		Parts: []*genai.Part{
			{Text: text},
		},
	}
}

func uniqueStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
