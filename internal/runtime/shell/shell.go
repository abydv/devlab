// Package shell implements the Shell Runtime: internal/runtime.Runtime
// backed directly by os/exec.
package shell

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/abydv/devlab/internal/runtime"
)

// Runtime executes commands directly via os/exec.
type Runtime struct{}

var _ runtime.Runtime = (*Runtime)(nil)

// New returns a Shell Runtime.
func New() *Runtime {
	return &Runtime{}
}

// Execute implements runtime.Runtime.
func (r *Runtime) Execute(ctx context.Context, cmd runtime.Command) (*runtime.Result, error) {
	if cmd.Name == "" {
		return nil, fmt.Errorf("shell: command name is required")
	}

	execCmd := exec.CommandContext(ctx, cmd.Name, cmd.Args...)
	execCmd.Dir = cmd.Dir
	if len(cmd.Env) > 0 {
		execCmd.Env = append(os.Environ(), cmd.Env...)
	}

	var stdout, stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	result := &runtime.Result{}

	var exitErr *exec.ExitError
	switch err := execCmd.Run(); {
	case err == nil:
		result.ExitCode = 0
	case ctx.Err() != nil:
		// The context was canceled or timed out: exec.CommandContext
		// kills the process, which surfaces as an *exec.ExitError
		// ("signal: killed") that would otherwise look like a normal
		// nonzero exit. Report it as the execution error it is.
		return nil, fmt.Errorf("shell: execute %s: %w", cmd.Name, ctx.Err())
	case errors.As(err, &exitErr):
		result.ExitCode = exitErr.ExitCode()
	default:
		return nil, fmt.Errorf("shell: execute %s: %w", cmd.Name, err)
	}

	result.Stdout = stdout.String()
	result.Stderr = stderr.String()
	return result, nil
}
