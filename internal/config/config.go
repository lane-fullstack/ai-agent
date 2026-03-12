package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const (
	configPath = "config.json"
)

var (
	mu           sync.RWMutex
	cachedConfig map[string]any
	lastModTime  time.Time
	watchOnce    sync.Once
)

func Load() map[string]any {
	watchOnce.Do(func() {
		cfg, modTime, err := loadConfigFile()
		if err != nil {
			log.Fatal("load config.json failed: ", err)
		}

		mu.Lock()
		cachedConfig = cfg
		lastModTime = modTime
		mu.Unlock()

		go watchConfigChanges()
	})

	mu.RLock()
	defer mu.RUnlock()
	return cloneMap(cachedConfig)
}

func watchConfigChanges() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Println("create config watcher failed:", err)
		return
	}
	defer watcher.Close()

	configDir := filepath.Dir(configPath)
	if configDir == "" {
		configDir = "."
	}
	configFilePath, err := filepath.Abs(configPath)
	if err != nil {
		log.Println("resolve config path failed:", err)
		return
	}

	if err = watcher.Add(configDir); err != nil {
		log.Println("watch config directory failed:", err)
		return
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			eventPath, err := filepath.Abs(event.Name)
			if err != nil {
				log.Println("resolve event path failed:", err)
				continue
			}

			if eventPath != configFilePath {
				continue
			}

			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Rename) {
				reloadConfig()
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("watch config.json failed:", err)
		}
	}
}

func reloadConfig() {
	cfg, modTime, err := loadConfigFile()
	if err != nil {
		log.Println("reload config.json failed:", err)
		return
	}

	mu.Lock()
	defer mu.Unlock()
	if !modTime.After(lastModTime) {
		return
	}

	cachedConfig = cfg
	lastModTime = modTime
	log.Println("config.json reloaded")
}

func loadConfigFile() (map[string]any, time.Time, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, time.Time{}, err
	}

	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, time.Time{}, err
	}

	applyDefaults(cfg)

	info, err := os.Stat(configPath)
	if err != nil {
		return nil, time.Time{}, err
	}

	return cfg, info.ModTime(), nil
}

func applyDefaults(cfg map[string]any) {
	if value, _ := cast[string](cfg["TranslateAPI"]); value == "" {
		cfg["TranslateAPI"] = "http://localhost:5050/translate"
	}
}

func Get[T any](key string) (T, error) {
	cfg := Load()
	return GetFrom[T](cfg, key)
}

func GetFrom[T any](cfg map[string]any, key string) (T, error) {
	value, ok := cfg[key]
	if !ok {
		var zero T
		return zero, fmt.Errorf("config key not found: %s", key)
	}

	return cast[T](value)
}

func cloneMap(src map[string]any) map[string]any {
	if src == nil {
		return map[string]any{}
	}

	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func AsString(value any) string {
	result, _ := cast[string](value)
	return result
}

func AsInt64Slice(value any) []int64 {
	result, _ := cast[[]int64](value)
	return result
}

func cast[T any](value any) (T, error) {
	var zero T
	if value == nil {
		return zero, fmt.Errorf("config value is nil")
	}

	if typed, ok := value.(T); ok {
		return typed, nil
	}

	switch any(zero).(type) {
	case string:
		if v, ok := value.(string); ok {
			return any(v).(T), nil
		}
	case bool:
		if v, ok := value.(bool); ok {
			return any(v).(T), nil
		}
	case int:
		switch v := value.(type) {
		case int:
			return any(v).(T), nil
		case int64:
			return any(int(v)).(T), nil
		case float64:
			return any(int(v)).(T), nil
		}
	case int64:
		switch v := value.(type) {
		case int:
			return any(int64(v)).(T), nil
		case int64:
			return any(v).(T), nil
		case float64:
			return any(int64(v)).(T), nil
		}
	case float64:
		switch v := value.(type) {
		case int:
			return any(float64(v)).(T), nil
		case int64:
			return any(float64(v)).(T), nil
		case float64:
			return any(v).(T), nil
		}
	case []int64:
		switch v := value.(type) {
		case []int64:
			return any(append([]int64(nil), v...)).(T), nil
		case []any:
			result := make([]int64, 0, len(v))
			for _, item := range v {
				n, err := cast[int64](item)
				if err != nil {
					return zero, err
				}
				result = append(result, n)
			}
			return any(result).(T), nil
		}
	case []string:
		switch v := value.(type) {
		case []string:
			return any(append([]string(nil), v...)).(T), nil
		case []any:
			result := make([]string, 0, len(v))
			for _, item := range v {
				s, err := cast[string](item)
				if err != nil {
					return zero, err
				}
				result = append(result, s)
			}
			return any(result).(T), nil
		}
	case map[string]any:
		if v, ok := value.(map[string]any); ok {
			return any(cloneMap(v)).(T), nil
		}
	}

	return zero, fmt.Errorf("config value type mismatch: want %T, got %T", zero, value)
}
