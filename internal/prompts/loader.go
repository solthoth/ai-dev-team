package prompts

import (
	"fmt"
	"os"
	"path/filepath"
)

func Load(dir, filename string) (string, error) {
	fullPath := filepath.Join(dir, filename)
	b, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("read prompt %s: %w", fullPath, err)
	}
	return string(b), nil
}