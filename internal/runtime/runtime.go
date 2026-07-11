// Package runtime defines the contract every Runtime implementation
// satisfies. A Runtime is the only layer permitted to execute
// operating system commands — Services must never call os/exec (or
// equivalent) directly, and instead depend on a Runtime.
package runtime

import "context"

// Command describes a single operating system command to execute.
type Command struct {
	// Name is the executable to run, e.g. "bash", "docker", "k3d".
	Name string
	// Args are the arguments passed to Name. Args are never
	// interpreted by a shell, so shell metacharacters have no special
	// meaning.
	Args []string
	// Dir is the working directory the command runs in. Empty means
	// the current process's working directory.
	Dir string
	// Env holds additional "KEY=VALUE" environment variables appended
	// to the current process's environment.
	Env []string
}

// Result is the outcome of successfully starting and running a Command
// to completion. A non-zero ExitCode is not itself an error — it is
// the command's own reported result.
type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

// Runtime executes operating system commands.
type Runtime interface {
	// Execute runs cmd to completion and returns its result. Execute
	// returns an error only when the command could not be started or
	// run to completion (e.g. the executable was not found, or ctx was
	// canceled) — never for a non-zero exit code.
	Execute(ctx context.Context, cmd Command) (*Result, error)
}
