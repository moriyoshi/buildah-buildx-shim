package cmd

import (
	"context"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/moriyoshi/buildah-buildx-shim/internal/buildahexec"
)

// builderName is the single synthetic builder this plugin reports. There is
// only ever one "builder" — the local buildah.
const builderName = "buildah"

// addStubCommands attaches the builder-lifecycle subcommands. Because buildah
// has no notion of separate builder instances, the lifecycle commands are
// no-ops (or thin reports) that keep buildx-aware tooling happy.
func addStubCommands(root *cobra.Command, ver string) {
	root.AddCommand(
		newVersionCmd(ver),
		newLsCmd(),
		newInspectCmd(),
		newNoopCmd("create", "Create a new builder instance", true),
		newNoopCmd("use", "Set the current builder instance", false),
		newNoopCmd("rm", "Remove one or more builder instances", false),
		newNoopCmd("stop", "Stop builder instance", false),
		newPruneCmd(),
		newDuCmd(),
	)
}

func newVersionCmd(ver string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show buildx version information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "github.com/moriyoshi/buildah-buildx-shim %s\n", ver)
			bv, err := buildahexec.Capture(cmd.Context(), "--version")
			if err == nil {
				fmt.Fprintf(cmd.OutOrStdout(), "backend: %s", bv)
			}
			return nil
		},
	}
}

func newLsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "List builder instances",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			platform := hostPlatform(cmd.Context())
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME/NODE\tDRIVER/ENDPOINT\tSTATUS\tBUILDKIT\tPLATFORMS")
			fmt.Fprintf(w, "%s *\t%s\t\t\t\n", builderName, builderName)
			fmt.Fprintf(w, "  %s\t%s\trunning\tn/a\t%s\n", builderName, builderName, platform)
			return w.Flush()
		},
	}
}

func newInspectCmd() *cobra.Command {
	var bootstrap bool
	cmd := &cobra.Command{
		Use:   "inspect [NAME]",
		Short: "Inspect current builder instance",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, _ []string) error {
			platform := hostPlatform(cmd.Context())
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Name:   %s\n", builderName)
			fmt.Fprintf(out, "Driver: %s\n", builderName)
			fmt.Fprintln(out, "Status: running")
			fmt.Fprintln(out)
			fmt.Fprintln(out, "Nodes:")
			fmt.Fprintf(out, "Name:      %s\n", builderName)
			fmt.Fprintln(out, "Status:    running")
			fmt.Fprintf(out, "Platforms: %s\n", platform)
			return nil
		},
	}
	cmd.Flags().BoolVar(&bootstrap, "bootstrap", false, "Ensure builder has booted before inspecting (no-op)")
	return cmd
}

// newNoopCmd returns a command that succeeds without doing anything. If echoName
// is set (used by `create`), it prints the builder name like buildx does.
func newNoopCmd(use, short string, echoName bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:                use,
		Short:              short,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if echoName {
				fmt.Fprintln(cmd.OutOrStdout(), builderName)
			}
			return nil
		},
	}
	return cmd
}

func newPruneCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "prune",
		Short:              "Remove build cache",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// buildah prune accepts a compatible flag set (-f/--force, --all).
			return runBuildah(cmd.Context(), append([]string{"prune"}, args...)...)
		},
	}
}

func newDuCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "du",
		Short: "Disk usage",
		RunE: func(cmd *cobra.Command, _ []string) error {
			warn(cmd, "du is not supported by the buildah backend; use `buildah images` / `podman system df`")
			return nil
		},
	}
}

// hostPlatform returns the host os/arch as reported by buildah, falling back to
// a blank string on error.
func hostPlatform(ctx context.Context) string {
	out, err := buildahexec.Capture(ctx, "info", "--format", "{{.host.os}}/{{.host.arch}}")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
