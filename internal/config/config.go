package config

import (
	"encoding/json"
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
	if AsString(cfg["TranslateAPI"]) == "" {
		cfg["TranslateAPI"] = "http://localhost:5050/translate"
	}
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
	if value == nil {
		return ""
	}

	if s, ok := value.(string); ok {
		return s
	}

	return ""
}

func AsInt64Slice(value any) []int64 {
	switch v := value.(type) {
	case []int64:
		return append([]int64(nil), v...)
	case []any:
		result := make([]int64, 0, len(v))
		for _, item := range v {
			switch n := item.(type) {
			case float64:
				result = append(result, int64(n))
			case int64:
				result = append(result, n)
			case int:
				result = append(result, int64(n))
			}
		}
		return result
	default:
		return nil
	}
}
