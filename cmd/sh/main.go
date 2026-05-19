// Package main is the entrypoint for the crush sh binary.
//
// It wraps the crush built-in bash tool's shell interpreter and exposes it as
// a standalone executable with two modes of operation:
//
//  1. Command mode:  sh -c <command>  — executes <command> and exits.
//  2. Interactive REPL mode (default when stdin is a terminal) — reads
//     commands line-by-line, maintains shell state (working directory,
//     environment variables) across commands, and provides tab completion
//     for file paths and commands found on PATH.
//
// Non-terminal stdin (pipes, redirects) is also handled: commands are read
// line-by-line and executed in order.
package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"

	xterm "golang.org/x/term"

	"github.com/charmbracelet/crush/internal/shell"
)

func main() {
	if err := run(); err != nil {
		var es interp.ExitStatus
		if errors.As(err, &es) {
			os.Exit(int(es))
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	args := os.Args[1:]

	// sh -c <command> [args...]
	if len(args) >= 2 && args[0] == "-c" {
		command := strings.Join(args[1:], " ")
		return runCommand(command)
	}

	// Non-interactive stdin (pipe or file redirect).
	if !xterm.IsTerminal(int(os.Stdin.Fd())) {
		return runFromReader(os.Stdin)
	}

	// Interactive REPL.
	return runInteractive()
}

// runCommand executes a single shell command string and returns.
func runCommand(command string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}
	return shell.Run(context.Background(), shell.RunOptions{
		Command: command,
		Cwd:     cwd,
		Env:     os.Environ(),
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	})
}

// runFromReader reads commands line-by-line from r and executes them,
// maintaining shell state (cwd, exported env vars) across calls.
func runFromReader(r io.Reader) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}
	sh := shell.NewShell(&shell.Options{
		WorkingDir: cwd,
		Env:        os.Environ(),
	})

	var buf strings.Builder
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		buf.WriteString(line)
		buf.WriteByte('\n')

		// Check whether we have a complete statement yet.
		src := buf.String()
		if _, parseErr := syntax.NewParser().Parse(strings.NewReader(src), ""); parseErr != nil {
			// Incomplete input — accumulate another line.
			if pe, ok := parseErr.(*syntax.ParseError); ok && pe.Incomplete {
				continue
			}
			// Real parse error — report and reset.
			fmt.Fprintln(os.Stderr, parseErr)
			buf.Reset()
			continue
		}

		if strings.TrimSpace(src) != "" {
			if execErr := sh.ExecStream(context.Background(), src, os.Stdout, os.Stderr); execErr != nil {
				if shell.IsInterrupt(execErr) {
					return execErr
				}
				// Mirror bash behaviour: print non-zero exits but keep running.
				exitCode := shell.ExitCode(execErr)
				if exitCode != 0 {
					fmt.Fprintf(os.Stderr, "exit status %d\n", exitCode)
				}
			}
		}
		buf.Reset()
	}
	return scanner.Err()
}

// stdinRW is a combined io.ReadWriter backed by stdin and stdout so that
// golang.org/x/term.Terminal can read keypresses and write echoed characters.
type stdinRW struct{}

func (stdinRW) Read(p []byte) (int, error)  { return os.Stdin.Read(p) }
func (stdinRW) Write(p []byte) (int, error) { return os.Stdout.Write(p) }

// runInteractive runs an interactive REPL.
func runInteractive() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}
	sh := shell.NewShell(&shell.Options{
		WorkingDir: cwd,
		Env:        os.Environ(),
	})

	// Switch the terminal into raw mode so term.Terminal controls echo and
	// line editing.
	fd := int(os.Stdin.Fd())
	oldState, err := xterm.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("switching terminal to raw mode: %w", err)
	}
	defer xterm.Restore(fd, oldState) //nolint:errcheck

	t := xterm.NewTerminal(stdinRW{}, buildPrompt(sh.GetWorkingDir()))
	t.AutoCompleteCallback = makeCompleter(sh)

	var buf strings.Builder
	for {
		line, readErr := t.ReadLine()
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return nil
			}
			return readErr
		}

		buf.WriteString(line)
		buf.WriteByte('\n')

		src := buf.String()

		// Check completeness of the accumulated input.
		_, parseErr := syntax.NewParser().Parse(strings.NewReader(src), "")
		if parseErr != nil {
			if pe, ok := parseErr.(*syntax.ParseError); ok && pe.Incomplete {
				// Need more input — show a continuation prompt.
				t.SetPrompt("> ")
				continue
			}
			// Real parse error.
			fmt.Fprintln(t, parseErr)
			buf.Reset()
			t.SetPrompt(buildPrompt(sh.GetWorkingDir()))
			continue
		}

		if strings.TrimSpace(src) != "" {
			// Restore normal terminal state so the child command can use
			// stdin/stdout directly (e.g. interactive sub-shells, editors).
			xterm.Restore(fd, oldState) //nolint:errcheck
			execErr := sh.ExecStream(context.Background(), src, os.Stdout, os.Stderr)
			// Re-enter raw mode after the command returns.
			if rawErr := restoreRaw(fd, &oldState); rawErr != nil {
				return rawErr
			}

			if execErr != nil {
				if shell.IsInterrupt(execErr) {
					return execErr
				}
				// Print non-zero exits (like bash -e would, but keep running).
				exitCode := shell.ExitCode(execErr)
				if exitCode != 0 {
					fmt.Fprintf(t, "exit status %d\r\n", exitCode)
				}
			}
		}

		buf.Reset()
		t.SetPrompt(buildPrompt(sh.GetWorkingDir()))
	}
}

// restoreRaw re-enters raw mode and updates *state.
func restoreRaw(fd int, state **xterm.State) error {
	s, err := xterm.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("re-entering raw terminal mode: %w", err)
	}
	*state = s
	return nil
}

// buildPrompt returns a shell prompt string that includes the shortened cwd.
func buildPrompt(cwd string) string {
	home, err := os.UserHomeDir()
	if err == nil && strings.HasPrefix(cwd, home) {
		cwd = "~" + cwd[len(home):]
	}
	return fmt.Sprintf("%s $ ", cwd)
}

// makeCompleter returns an AutoCompleteCallback for golang.org/x/term.Terminal
// that completes file paths and, for the first token on a line, also commands
// found on PATH.
func makeCompleter(sh *shell.Shell) func(line string, pos int, key rune) (string, int, bool) {
	return func(line string, pos int, key rune) (string, int, bool) {
		if key != '\t' {
			return "", 0, false
		}

		// Determine the word being completed (up to cursor position).
		prefix := line[:pos]
		word := currentWord(prefix)

		isFirstWord := isFirstToken(prefix, word)

		var candidates []string

		// File/directory completions are always relevant.
		fileCandidates := completeFiles(word, sh.GetWorkingDir())
		candidates = append(candidates, fileCandidates...)

		// Command completions for the first token (unless word looks like a path).
		if isFirstWord && !looksLikePath(word) {
			cmdCandidates := completeCommands(word)
			candidates = append(candidates, cmdCandidates...)
		}

		candidates = dedup(candidates)
		sort.Strings(candidates)

		if len(candidates) == 0 {
			return "", 0, false
		}

		// Complete to longest common prefix.
		common := longestCommonPrefix(candidates)
		if len(common) <= len(word) {
			// Nothing new to add; don't consume the tab.
			return "", 0, false
		}

		// Build the new line by replacing the current word with common.
		newLine := line[:pos-len(word)] + common + line[pos:]
		newPos := pos - len(word) + len(common)
		return newLine, newPos, true
	}
}

// currentWord extracts the token being typed up to the cursor.
func currentWord(prefix string) string {
	// Walk backward from end to find the start of the current token.
	// Tokens are separated by unquoted spaces; we keep it simple here.
	for i := len(prefix) - 1; i >= 0; i-- {
		if prefix[i] == ' ' || prefix[i] == '\t' {
			return prefix[i+1:]
		}
	}
	return prefix
}

// isFirstToken reports whether word is the first token on the line.
func isFirstToken(prefix, word string) bool {
	before := strings.TrimRight(prefix[:len(prefix)-len(word)], " \t")
	return before == ""
}

// looksLikePath reports whether word begins with a path prefix.
func looksLikePath(word string) bool {
	return strings.HasPrefix(word, "/") ||
		strings.HasPrefix(word, "./") ||
		strings.HasPrefix(word, "../") ||
		strings.HasPrefix(word, "~")
}

// completeFiles returns file-system entries that match prefix relative to cwd.
func completeFiles(prefix, cwd string) []string {
	dir, file := filepath.Split(prefix)
	if dir == "" {
		dir = "."
	} else if strings.HasPrefix(dir, "~/") {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, dir[2:])
	}

	if !filepath.IsAbs(dir) {
		dir = filepath.Join(cwd, dir)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var out []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, file) {
			continue
		}
		candidate := filepath.Join(filepath.Dir(prefix), name)
		if filepath.Dir(prefix) == "." && !strings.Contains(prefix, "/") {
			candidate = name
		}
		if e.IsDir() {
			candidate += "/"
		}
		out = append(out, candidate)
	}
	return out
}

// completeCommands returns executables on PATH whose name starts with prefix.
func completeCommands(prefix string) []string {
	seen := make(map[string]struct{})
	var out []string

	pathEnv := os.Getenv("PATH")
	for _, dir := range filepath.SplitList(pathEnv) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			name := e.Name()
			if _, already := seen[name]; already {
				continue
			}
			if !strings.HasPrefix(name, prefix) {
				continue
			}
			// Only include executable files.
			info, err := e.Info()
			if err != nil {
				continue
			}
			if info.Mode()&0o111 == 0 {
				continue
			}
			// Skip directories that happen to be executable.
			if info.IsDir() {
				continue
			}
			// On non-Unix systems a file might not have x bit set in
			// the fs.FileMode; fall back to LookPath as a check.
			if _, lookErr := exec.LookPath(filepath.Join(dir, name)); lookErr != nil {
				if !errors.Is(lookErr, fs.ErrPermission) {
					continue
				}
			}
			seen[name] = struct{}{}
			out = append(out, name)
		}
	}
	return out
}

// longestCommonPrefix returns the longest string that all candidates share as
// a prefix.
func longestCommonPrefix(candidates []string) string {
	if len(candidates) == 0 {
		return ""
	}
	prefix := candidates[0]
	for _, c := range candidates[1:] {
		for !strings.HasPrefix(c, prefix) {
			prefix = prefix[:len(prefix)-1]
			if prefix == "" {
				return ""
			}
		}
	}
	return prefix
}

// dedup removes duplicate strings while preserving order.
func dedup(ss []string) []string {
	seen := make(map[string]struct{}, len(ss))
	out := ss[:0]
	for _, s := range ss {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}
