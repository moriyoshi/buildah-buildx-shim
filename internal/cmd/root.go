// Package cmd holds the cobra command tree for the buildx-compatible plugin.
package cmd

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/spf13/cobra"

	"github.com/moriyoshi/buildah-buildx-shim/internal/buildahexec"
)

// NewRootCmd builds the root `buildx` command and attaches every subcommand.
// name is the plugin name ("buildx"); ver is the plugin version string.
func NewRootCmd(name, ver string) *cobra.Command {
	root := &cobra.Command{
		Use:   name,
		Short: "Build images with buildah using the docker buildx interface",
		Long: "buildah-backed implementation of docker buildx. Most build flags map\n" +
			"directly to buildah; see the project README for unsupported flags.",
		// Plugins should not error out with usage text for runtime failures.
		SilenceUsage: true,
	}

	root.AddCommand(
		newBuildCmd(),
		newBakeCmd(),
		newImagetoolsCmd(),
	)
	addStubCommands(root, ver)

	return root
}

// runBuildah runs a buildah command, mapping a non-zero exit into a
// cli.StatusError so the plugin framework propagates buildah's exit code.
func runBuildah(ctx context.Context, args ...string) error {
	if err := buildahexec.Run(ctx, args...); err != nil {
		return buildahStatus(err, firstArg(args))
	}
	return nil
}

// buildahStatus converts a buildah exec error into a cli.StatusError carrying
// the child's exit code. buildah has already written its own diagnostics to
// stderr, so the Status message is just a concise summary.
func buildahStatus(err error, verb string) error {
	if err == nil {
		return nil
	}
	code := buildahexec.ExitCode(err)
	return cli.StatusError{
		StatusCode: code,
		Status:     fmt.Sprintf("buildah %s: exit status %d", verb, code),
	}
}

func firstArg(args []string) string {
	if len(args) == 0 {
		return "build"
	}
	return args[0]
}

// warn prints a non-fatal message about a buildx feature we can't honour.
func warn(cmd *cobra.Command, format string, a ...any) {
	fmt.Fprintf(cmd.ErrOrStderr(), "buildx(buildah): "+format+"\n", a...)
}
