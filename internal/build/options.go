// Package build translates `docker buildx build` invocations into one or more
// buildah commands.
package build

import (
	"strings"

	"github.com/spf13/pflag"
)

// Options mirrors the flag surface of `docker buildx build`. Fields we can map
// to buildah are translated; the rest are dropped with a warning (see
// translate.go / unsupported.go).
type Options struct {
	// Context is the positional build-context argument (PATH | URL | -).
	Context string

	// Passthrough-ish flags (mostly 1:1 with buildah build).
	Tags          []string
	File          string
	BuildArgs     []string
	BuildContexts []string
	Target        string
	Platforms     []string
	NoCache       bool
	Pull          bool
	Labels        []string
	Annotations   []string
	Network       string
	AddHosts      []string
	CacheFrom     []string
	CacheTo       []string
	CgroupParent  string
	ShmSize       string
	Ulimits       []string
	Secrets       []string
	SSH           []string
	Quiet         bool
	IIDFile       string

	// Output / destination flags.
	Push    bool
	Load    bool
	Outputs []string

	// Metadata.
	MetadataFile string

	// Flags accepted for compatibility but unsupported by the buildah backend.
	Allow         []string
	Attests       []string
	NoCacheFilter []string
	Progress      string
	Provenance    string
	SBOM          string
	Builder       string
}

// AddFlags registers every buildx build flag on the given flag set, binding
// into o. Flag names and shorthands match docker buildx so `--help` and
// scripted invocations behave the same.
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringArrayVarP(&o.Tags, "tag", "t", nil, "Name and optionally a tag (format: name:tag)")
	fs.StringVarP(&o.File, "file", "f", "", "Name of the Dockerfile (default: PATH/Dockerfile)")
	fs.StringArrayVar(&o.BuildArgs, "build-arg", nil, "Set build-time variables")
	fs.StringArrayVar(&o.BuildContexts, "build-context", nil, "Additional build contexts (e.g., name=path)")
	fs.StringVar(&o.Target, "target", "", "Set the target build stage to build")
	fs.StringArrayVar(&o.Platforms, "platform", nil, "Set target platform(s) for build")
	fs.BoolVar(&o.NoCache, "no-cache", false, "Do not use cache when building the image")
	fs.BoolVar(&o.Pull, "pull", false, "Always attempt to pull all referenced images")
	fs.StringArrayVar(&o.Labels, "label", nil, "Set metadata for an image")
	fs.StringArrayVar(&o.Annotations, "annotation", nil, "Add annotation to the image")
	fs.StringVar(&o.Network, "network", "", "Set the networking mode for the RUN instructions during build")
	fs.StringSliceVar(&o.AddHosts, "add-host", nil, "Add a custom host-to-IP mapping (format: host:ip)")
	fs.StringArrayVar(&o.CacheFrom, "cache-from", nil, "External cache sources")
	fs.StringArrayVar(&o.CacheTo, "cache-to", nil, "Cache export destinations")
	fs.StringVar(&o.CgroupParent, "cgroup-parent", "", "Set the parent cgroup for the RUN instructions during build")
	fs.StringVar(&o.ShmSize, "shm-size", "", "Size of /dev/shm")
	fs.StringArrayVar(&o.Ulimits, "ulimit", nil, "Ulimit options")
	fs.StringArrayVar(&o.Secrets, "secret", nil, "Secret to expose to the build")
	fs.StringArrayVar(&o.SSH, "ssh", nil, "SSH agent socket or keys to expose to the build")
	fs.BoolVarP(&o.Quiet, "quiet", "q", false, "Suppress the build output and print image ID on success")
	fs.StringVar(&o.IIDFile, "iidfile", "", "Write the image ID to a file")

	fs.BoolVar(&o.Push, "push", false, "Shorthand for --output=type=registry")
	fs.BoolVar(&o.Load, "load", false, "Shorthand for --output=type=docker")
	fs.StringArrayVarP(&o.Outputs, "output", "o", nil, "Output destination (format: type=local,dest=path)")

	fs.StringVar(&o.MetadataFile, "metadata-file", "", "Write build result metadata to a file")

	// Compatibility-only flags (parsed, then dropped with a warning).
	fs.StringSliceVar(&o.Allow, "allow", nil, "Allow extra privileged entitlement (unsupported)")
	fs.StringArrayVar(&o.Attests, "attest", nil, "Attestation parameters (unsupported)")
	fs.StringSliceVar(&o.NoCacheFilter, "no-cache-filter", nil, "Do not cache specified stages (unsupported)")
	fs.StringVar(&o.Progress, "progress", "auto", "Set type of progress output (ignored)")
	fs.StringVar(&o.Provenance, "provenance", "", "Provenance attestation (unsupported)")
	fs.StringVar(&o.SBOM, "sbom", "", "SBOM attestation (unsupported)")
	fs.StringVar(&o.Builder, "builder", "", "Override the configured builder instance (ignored)")
}

// platformCount returns the total number of target platforms across all
// --platform entries (which may themselves be comma-separated).
func (o *Options) platformCount() int {
	n := 0
	for _, p := range o.Platforms {
		for _, part := range strings.Split(p, ",") {
			if strings.TrimSpace(part) != "" {
				n++
			}
		}
	}
	return n
}

// MultiPlatform reports whether more than one target platform was requested.
func (o *Options) MultiPlatform() bool { return o.platformCount() > 1 }
