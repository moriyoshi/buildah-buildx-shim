package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/moriyoshi/buildah-buildx-shim/internal/bake"
	"github.com/moriyoshi/buildah-buildx-shim/internal/build"
)

func newBakeCmd() *cobra.Command {
	var (
		files        []string
		sets         []string
		print        bool
		push         bool
		load         bool
		metadataFile string
		progress     string
		allow        []string
	)
	cmd := &cobra.Command{
		Use:   "bake [OPTIONS] [TARGET...]",
		Short: "Build from a file",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = progress // accepted for compose compatibility; buildah has no equivalent
			_ = allow    // BuildKit entitlements; not needed by the buildah backend

			cfg, err := bake.Load(files)
			if err != nil {
				return err
			}
			targets, err := bake.Resolve(cfg, args)
			if err != nil {
				return err
			}
			if err := bake.ApplyOverrides(targets, sets); err != nil {
				return err
			}

			if print {
				return printBake(cmd, args, cfg, targets)
			}

			metaByTarget := map[string]map[string]any{}
			for _, t := range targets {
				o := targetToOptions(t, push, load)
				if err := buildBakeTarget(cmd, t.Name, o, metadataFile != "", metaByTarget); err != nil {
					return err
				}
			}
			if metadataFile != "" {
				if err := build.WriteBakeMetadataFile(metadataFile, metaByTarget); err != nil {
					warn(cmd, "failed to write metadata file: %v", err)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringArrayVarP(&files, "file", "f", nil, "Build definition file")
	cmd.Flags().StringArrayVar(&sets, "set", nil, "Override target value (e.g., targetpattern.key=value)")
	cmd.Flags().BoolVar(&print, "print", false, "Print the options without building")
	cmd.Flags().BoolVar(&push, "push", false, "Shorthand for --set=*.output=type=registry")
	cmd.Flags().BoolVar(&load, "load", false, "Shorthand for --set=*.output=type=docker")
	cmd.Flags().StringVar(&metadataFile, "metadata-file", "", "Write build result metadata to a file")
	cmd.Flags().StringVar(&progress, "progress", "auto", "Set type of progress output (ignored)")
	cmd.Flags().StringArrayVar(&allow, "allow", nil, "Allow build to access specified resources (ignored)")
	return cmd
}

// buildBakeTarget builds one resolved bake target. When metadata is requested,
// it captures the image ID via a temporary iidfile and records the per-target
// metadata entry.
func buildBakeTarget(cmd *cobra.Command, name string, o *build.Options, wantMeta bool, out map[string]map[string]any) error {
	warn(cmd, "building target %q", name)
	var iidPath string
	if wantMeta && o.IIDFile == "" {
		tmp, err := os.CreateTemp("", "bake-iid-*")
		if err != nil {
			return err
		}
		iidPath = tmp.Name()
		tmp.Close()
		defer os.Remove(iidPath)
		o.IIDFile = iidPath
	}
	if err := runBuild(cmd, o); err != nil {
		return err
	}
	if wantMeta {
		primary := ""
		if len(o.Tags) > 0 {
			primary = o.Tags[0]
		}
		out[name] = build.Metadata(o.IIDFile, primary)
	}
	return nil
}

func targetToOptions(t *bake.Target, push, load bool) *build.Options {
	o := &build.Options{Context: "."}
	if t.Context != nil {
		o.Context = *t.Context
	}
	if t.Dockerfile != nil {
		o.File = *t.Dockerfile
	}
	if t.Target != nil {
		o.Target = *t.Target
	}
	o.Tags = t.Tags
	o.Platforms = t.Platforms
	o.BuildArgs = mapToKV(t.Args)
	o.Labels = mapToKV(t.Labels)
	o.Secrets = t.Secret
	o.SSH = t.SSH
	o.CacheFrom = t.CacheFrom
	o.CacheTo = t.CacheTo
	o.Outputs = t.Output
	if t.NoCache != nil {
		o.NoCache = *t.NoCache
	}
	if t.Pull != nil {
		o.Pull = *t.Pull
	}
	o.Push = push
	o.Load = load
	return o
}

// mapToKV converts a bake args/labels map into the "key=value" form buildah
// expects, with stable ordering for reproducibility.
func mapToKV(m map[string]string) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(m))
	for _, k := range keys {
		out = append(out, fmt.Sprintf("%s=%s", k, m[k]))
	}
	return out
}

// printBake emits the resolved configuration as JSON, mirroring `buildx bake
// --print`.
func printBake(cmd *cobra.Command, requested []string, cfg *bake.Config, targets []*bake.Target) error {
	names := make([]string, 0, len(targets))
	tmap := map[string]any{}
	for _, t := range targets {
		names = append(names, t.Name)
		tmap[t.Name] = t
	}
	groupTargets := requested
	if len(groupTargets) == 0 {
		groupTargets = names
	}
	doc := map[string]any{
		"group":  map[string]any{"default": map[string]any{"targets": groupTargets}},
		"target": tmap,
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(b))
	return nil
}
