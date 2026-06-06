package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/moriyoshi/buildah-buildx-shim/internal/build"
)

func newBuildCmd() *cobra.Command {
	o := &build.Options{}
	cmd := &cobra.Command{
		Use:     "build [OPTIONS] PATH | URL | -",
		Aliases: []string{"b"},
		Short:   "Start a build",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Context = "."
			if len(args) == 1 {
				o.Context = args[0]
			}
			return runBuild(cmd, o)
		},
	}
	o.AddFlags(cmd.Flags())
	return cmd
}

func runBuild(cmd *cobra.Command, o *build.Options) error {
	ctx := cmd.Context()
	warnf := func(format string, a ...any) { warn(cmd, format, a...) }

	plan, err := build.ResolveOutput(o, warnf)
	if err != nil {
		return err
	}

	// If a metadata file was requested but the user didn't ask for an iidfile,
	// capture the image ID into a temp file so we can synthesise metadata.
	iidPath := o.IIDFile
	if o.MetadataFile != "" && iidPath == "" {
		tmp, terr := os.CreateTemp("", "buildx-iid-*")
		if terr != nil {
			return terr
		}
		iidPath = tmp.Name()
		tmp.Close()
		defer os.Remove(iidPath)
	}

	buildArgs := []string{"build"}
	buildArgs = append(buildArgs, build.CommonArgs(o, warnf)...)
	buildArgs = append(buildArgs, plan.BuildArgs...)
	if iidPath != "" {
		buildArgs = append(buildArgs, "--iidfile", iidPath)
	}
	buildArgs = append(buildArgs, o.Context)

	if err := runBuildah(ctx, buildArgs...); err != nil {
		return err
	}

	for _, step := range plan.Post {
		if err := runBuildah(ctx, step.Args...); err != nil {
			return err
		}
	}

	if o.MetadataFile != "" {
		if err := build.WriteMetadataFile(o.MetadataFile, iidPath, plan.PrimaryRef); err != nil {
			warnf("failed to write metadata file: %v", err)
		}
	}
	return nil
}
