// Command docker-buildx is a Docker CLI plugin that implements the
// `docker buildx` command surface on top of buildah instead of BuildKit.
//
// Install it by placing (or symlinking) the built binary at
// ~/.docker/cli-plugins/docker-buildx (or the system cli-plugins dir).
package main

import (
	"errors"
	"os"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli-plugins/metadata"
	"github.com/docker/cli/cli-plugins/plugin"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"

	"github.com/moriyoshi/buildah-buildx-shim/internal/cmd"
)

func main() {
	// When invoked directly (e.g. `docker-buildx bake ...` by docker compose)
	// rather than re-execed by the docker CLI, run the root command standalone
	// so build/bake/imagetools are top-level subcommands. Otherwise the plugin
	// framework expects the "buildx" plugin name as the first argument.
	if plugin.RunningStandalone() {
		runStandalone()
		return
	}

	plugin.Run(func(dockerCli command.Cli) *cobra.Command {
		return cmd.NewRootCmd("buildx", version)
	}, metadata.Metadata{
		SchemaVersion: "0.1.0",
		Vendor:        "moriyoshi",
		// Advertise a buildx-compatible version (not the project version) so
		// buildx-gated tooling such as `docker compose build` accepts the shim.
		Version:          buildxCompatVersion,
		ShortDescription: "Docker Buildx (buildah backend)",
		URL:              "https://github.com/moriyoshi/buildah-buildx-shim",
	})
}

func runStandalone() {
	root := cmd.NewRootCmd("buildx", version)
	if err := root.Execute(); err != nil {
		var st cli.StatusError
		if errors.As(err, &st) && st.StatusCode != 0 {
			os.Exit(st.StatusCode)
		}
		os.Exit(1)
	}
}
