package executor

import (
	"ai-agent/internal/model"
	"bytes"
	"errors"
	"os/exec"
	"sync"
)

type handle func(taskId int64) (string, error)

var mu sync.Mutex
var InternalMap = make(map[string]handle)

func RegisterFunc(name string, f handle) {
	mu.Lock()
	defer mu.Unlock()
	InternalMap[name] = f
}

func Run(task model.Task) (string, error) {
	taskType := task.Type
	command := task.Command
	switch taskType {

	case "bash":
		cmd := exec.Command("bash", "-c", command)
		var out bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		err := cmd.Run()
		return out.String() + stderr.String(), err

	case "python":
		cmd := exec.Command("python3", command)
		var out bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		err := cmd.Run()
		return out.String() + stderr.String(), err

	case "binary":
		cmd := exec.Command(command)
		var out bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		err := cmd.Run()
		return out.String() + stderr.String(), err

	case "internal":
		// Handle internal Go functions
		f, ok := InternalMap[command]
		if !ok {
			return "", errors.New("internal function not found")
		}
		return f(task.ID)

	default:
		cmd := exec.Command("bash", "-c", command)
		var out bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		err := cmd.Run()
		return out.String() + stderr.String(), err
	}
}
