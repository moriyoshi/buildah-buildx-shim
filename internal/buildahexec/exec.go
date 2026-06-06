// Package buildahexec locates and runs the buildah binary on behalf of the
// plugin. Every buildx subcommand ultimately turns into one or more buildah
// invocations, so all process spawning is funnelled through here.
package buildahexec

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
)

// BinaryEnv lets users point the wrapper at a specific buildah binary, e.g.
// when it is not on PATH or they want to pin a version.
const BinaryEnv = "BUILDAH_BINARY"

// Binary returns the buildah executable path, honouring $BUILDAH_BINARY and
// otherwise resolving "buildah" on PATH.
func Binary() (string, error) {
	if env := os.Getenv(BinaryEnv); env != "" {
		return env, nil
	}
	p, err := exec.LookPath("buildah")
	if err != nil {
		return "", fmt.Errorf("buildah not found on PATH (set %s to override): %w", BinaryEnv, err)
	}
	return p, nil
}

// Run executes buildah with the given arguments, wiring the child's stdio to
// the current process. The returned error preserves the child's exit code (see
// ExitCode) so callers can propagate it.
func Run(ctx context.Context, args ...string) error {
	return RunWithStdin(ctx, os.Stdin, args...)
}

// RunWithStdin is like Run but lets the caller supply stdin (used for the
// `build -` / Dockerfile-on-stdin cases).
func RunWithStdin(ctx context.Context, stdin *os.File, args ...string) error {
	bin, err := Binary()
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stdin = stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Capture runs buildah and returns its stdout. Stderr is passed through to the
// user so progress/errors remain visible. Used for steps whose output the
// plugin parses (manifest inspect, info).
func Capture(ctx context.Context, args ...string) ([]byte, error) {
	bin, err := Binary()
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stderr = os.Stderr
	return cmd.Output()
}

// ExitCode extracts the process exit code from an error returned by Run. It
// returns 0 for nil, and 1 for non-exec errors.
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		if code := ee.ExitCode(); code >= 0 {
			return code
		}
	}
	return 1
}
