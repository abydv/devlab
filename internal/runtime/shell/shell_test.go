package shell

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/abydv/devlab/internal/runtime"
)

func TestExecuteCapturesStdout(t *testing.T) {
	r := New()

	result, err := r.Execute(context.Background(), runtime.Command{
		Name: "echo",
		Args: []string{"hello"},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if strings.TrimSpace(result.Stdout) != "hello" {
		t.Errorf("Stdout = %q, want %q", result.Stdout, "hello")
	}
}

func TestExecuteCapturesNonZeroExitCode(t *testing.T) {
	r := New()

	result, err := r.Execute(context.Background(), runtime.Command{
		Name: "sh",
		Args: []string{"-c", "exit 3"},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.ExitCode != 3 {
		t.Errorf("ExitCode = %d, want 3", result.ExitCode)
	}
}

func TestExecuteCapturesStderr(t *testing.T) {
	r := New()

	result, err := r.Execute(context.Background(), runtime.Command{
		Name: "sh",
		Args: []string{"-c", "echo oops >&2"},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if strings.TrimSpace(result.Stderr) != "oops" {
		t.Errorf("Stderr = %q, want %q", result.Stderr, "oops")
	}
}

func TestExecuteUsesWorkingDirectory(t *testing.T) {
	r := New()
	dir := t.TempDir()

	result, err := r.Execute(context.Background(), runtime.Command{
		Name: "pwd",
		Dir:  dir,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if got := strings.TrimSpace(result.Stdout); got != dir {
		t.Errorf("pwd = %q, want %q", got, dir)
	}
}

func TestExecuteAppliesEnv(t *testing.T) {
	r := New()

	result, err := r.Execute(context.Background(), runtime.Command{
		Name: "sh",
		Args: []string{"-c", "echo $DEVLAB_TEST_VAR"},
		Env:  []string{"DEVLAB_TEST_VAR=configured"},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if got := strings.TrimSpace(result.Stdout); got != "configured" {
		t.Errorf("Stdout = %q, want %q", got, "configured")
	}
}

func TestExecuteRequiresName(t *testing.T) {
	r := New()

	if _, err := r.Execute(context.Background(), runtime.Command{}); err == nil {
		t.Fatal("Execute() error = nil, want an error for empty command name")
	}
}

func TestExecuteUnknownCommand(t *testing.T) {
	r := New()

	if _, err := r.Execute(context.Background(), runtime.Command{Name: "devlab-does-not-exist-xyz"}); err == nil {
		t.Fatal("Execute() error = nil, want an error for a missing executable")
	}
}

func TestExecuteRespectsContextCancellation(t *testing.T) {
	r := New()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	if _, err := r.Execute(ctx, runtime.Command{Name: "sleep", Args: []string{"5"}}); err == nil {
		t.Fatal("Execute() error = nil, want an error when the context is canceled")
	}
}
