package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/moriyoshi/buildah-buildx-shim/internal/buildahexec"
)

func newImagetoolsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "imagetools",
		Short: "Commands to work on images in registry",
	}
	cmd.AddCommand(newImagetoolsInspectCmd(), newImagetoolsCreateCmd())
	return cmd
}

func newImagetoolsInspectCmd() *cobra.Command {
	var (
		raw    bool
		format string
	)
	cmd := &cobra.Command{
		Use:   "inspect [OPTIONS] NAME",
		Short: "Show details of an image in the registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if format != "" {
				warn(cmd, "--format templates are not supported by the buildah backend; printing raw manifest")
			}
			_ = raw // both default and --raw emit the manifest JSON buildah returns
			out, err := buildahexec.Capture(cmd.Context(), "manifest", "inspect", args[0])
			if err != nil {
				// Capture already streamed buildah's stderr; just propagate the code.
				return buildahStatus(err, "manifest inspect")
			}
			cmd.OutOrStdout().Write(out)
			return nil
		},
	}
	cmd.Flags().BoolVar(&raw, "raw", false, "Show original, unprocessed JSON manifest")
	cmd.Flags().StringVar(&format, "format", "", "Format the output (limited support)")
	return cmd
}

func newImagetoolsCreateCmd() *cobra.Command {
	var (
		tags   []string
		append bool
		dryRun bool
	)
	cmd := &cobra.Command{
		Use:   "create [OPTIONS] [SOURCE...]",
		Short: "Create a new image based on source images",
		RunE: func(cmd *cobra.Command, sources []string) error {
			if len(tags) == 0 {
				return fmt.Errorf("at least one -t/--tag is required")
			}
			ctx := cmd.Context()
			list := tags[0]

			if dryRun {
				fmt.Fprintf(cmd.OutOrStdout(), "would create manifest %s from %v\n", list, sources)
				return nil
			}

			if !append {
				// Remove any stale list of this name, ignoring failure, then create.
				_ = buildahexec.Run(ctx, "manifest", "rm", list)
				if err := runBuildah(ctx, "manifest", "create", list); err != nil {
					return err
				}
			}
			for _, src := range sources {
				if err := runBuildah(ctx, "manifest", "add", list, src); err != nil {
					return err
				}
			}
			for _, t := range tags {
				if err := runBuildah(ctx, "manifest", "push", "--all", list, "docker://"+t); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().StringArrayVarP(&tags, "tag", "t", nil, "Set reference for new image")
	cmd.Flags().BoolVar(&append, "append", false, "Append to existing manifest")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show final image instead of pushing")
	return cmd
}
