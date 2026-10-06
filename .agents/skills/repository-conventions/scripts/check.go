package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

func forbidden(p string) bool {
	p = strings.ToLower(strings.ReplaceAll(p, "\\", "/"))
	if strings.HasPrefix(p, "vendor/") || strings.HasPrefix(p, "node_modules/") {
		return false
	}
	for _, part := range strings.Split(p, "/") {
		switch part {
		case ".cursor", ".claude", ".windsurf", ".gemini", ".codex", ".clinerules", ".roo", ".junie":
			return true
		}
	}
	base := path.Base(p)
	if strings.HasSuffix(base, ".mdc") || base == "claude.md" || base == "gemini.md" || base == ".cursorrules" || base == ".windsurfrules" || base == "copilot-instructions.md" {
		return true
	}
	return strings.Contains(p, ".github/instructions/") || strings.HasSuffix(p, "/agents/openai.yaml")
}
func paths(dir string) ([]string, error) {
	cmd := exec.Command("git", "ls-files", "-z", "--cached", "--others", "--exclude-standard")
	cmd.Dir = dir
	if dir != "" {
		cmd.Env = isolatedGitEnv()
	}
	b, e := cmd.Output()
	if e != nil {
		return nil, e
	}
	var result []string
	for _, p := range bytes.Split(b, []byte{0}) {
		if len(p) > 0 {
			result = append(result, string(p))
		}
	}
	return result, nil
}

// Самопроверка не должна наследовать index/worktree/config вызывающего hook.
// Обычная проверка paths("") сохраняет окружение настоящего staged index.
func isolatedGitEnv() []string {
	var env []string
	for _, v := range os.Environ() {
		if !strings.HasPrefix(v, "GIT_") {
			env = append(env, v)
		}
	}
	return append(env, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull)
}

// Проверяем настоящий index, включая staged deletion и ещё не добавленные файлы.
func indexTest() error {
	dir, err := os.MkdirTemp("", "portable-instructions-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	run := func(args ...string) error {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = isolatedGitEnv()
		return cmd.Run()
	}
	if err = run("init", "-q"); err != nil {
		return err
	}
	p := filepath.Join(dir, "rule.mdc")
	if err = os.WriteFile(p, []byte("rule\n"), 0600); err != nil {
		return err
	}
	for _, staged := range []bool{false, true} {
		if staged {
			if err = run("add", "rule.mdc"); err != nil {
				return err
			}
		}
		files, e := paths(dir)
		if e != nil {
			return e
		}
		if len(files) != 1 || !forbidden(files[0]) {
			return fmt.Errorf("index did not detect forbidden path, staged=%v", staged)
		}
	}
	if err = os.Remove(p); err != nil {
		return err
	}
	if err = run("add", "-u"); err != nil {
		return err
	}
	files, err := paths(dir)
	if err != nil {
		return err
	}
	if len(files) != 0 {
		return fmt.Errorf("staged deletion still rejected: %v", files)
	}
	return nil
}
func main() {
	self := flag.Bool("self-test", false, "проверить запрет и переносимые пути")
	flag.Parse()
	if *self {
		cases := map[string]bool{"AGENTS.md": false, ".agents/skills/test/SKILL.md": false, "docs/agents.md": false, ".github/workflows/check.yml": false, "vendor/lib/CLAUDE.md": false, "app/.cursor/rules/x.mdc": true, "CLAUDE.md": true, "x/GEMINI.md": true, "x/.clinerules/rule.md": true, ".claude/skills": true, ".agents/skills/x/agents/openai.yaml": true, "rules/My Rule.MDC": true, ".github/instructions/go.instructions.md": true}
		for p, want := range cases {
			if forbidden(p) != want {
				fmt.Fprintln(os.Stderr, "ошибка проверки:", p)
				os.Exit(1)
			}
		}
		if err := indexTest(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("Portable agent policy self-test passed (paths, untracked, index, staged deletion)")
		return
	}
	files, e := paths("")
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
	seen := map[string]bool{}
	for _, p := range files {
		if forbidden(p) && !seen[p] {
			fmt.Fprintln(os.Stderr, "Непереносимые инструкции:", p, "→ используйте AGENTS.md / .agents/skills")
			seen[p] = true
		}
	}
	if len(seen) > 0 {
		os.Exit(1)
	}
	fmt.Println("Portable agent instructions verified")
}
