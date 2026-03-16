package contextpack

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type PackOptions struct {
	RepoPath string
}

func Build(opts PackOptions) (string, error) {
	repoPath := strings.TrimSpace(opts.RepoPath)
	if repoPath == "" {
		return "", fmt.Errorf("repo path is required")
	}

	var b strings.Builder

	filesOut, _ := run(repoPath, "git", "ls-files")
	statusOut, _ := run(repoPath, "git", "status", "--short")
	diffOut, _ := run(repoPath, "git", "diff", "--", ".")

	readmeOut := readIfExists(
		filepath.Join(repoPath, "README.md"),
		filepath.Join(repoPath, "readme.md"),
	)

	if strings.TrimSpace(filesOut) != "" {
		b.WriteString("## REPO FILES\n")
		b.WriteString(filesOut)
		b.WriteString("\n\n")
	}

	if strings.TrimSpace(readmeOut) != "" {
		b.WriteString("## README\n")
		b.WriteString(readmeOut)
		b.WriteString("\n\n")
	}

	if strings.TrimSpace(statusOut) != "" {
		b.WriteString("## GIT STATUS\n")
		b.WriteString(statusOut)
		b.WriteString("\n\n")
	}

	if strings.TrimSpace(diffOut) != "" {
		b.WriteString("## CURRENT DIFF\n")
		b.WriteString(diffOut)
		b.WriteString("\n\n")
	}

	return b.String(), nil
}

func readIfExists(paths ...string) string {
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err == nil {
			return string(b)
		}
	}
	return ""
}

func run(dir string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s %v failed: %w: %s", name, args, err, stderr.String())
	}

	return stdout.String(), nil
}