package build

import (
	"fmt"
	"strings"
)

// Plan describes how to satisfy a build's requested outputs: extra arguments to
// append to `buildah build`, plus buildah commands to run afterwards (pushing
// tags, exporting archives, etc.).
type Plan struct {
	// BuildArgs are appended to the `buildah build` invocation (tags via -t or,
	// for multi-platform, --manifest, plus any filesystem -o exports).
	BuildArgs []string
	// Post are buildah invocations to run, in order, after a successful build.
	Post []PostStep
	// PrimaryRef is the local image/manifest reference the build produces; used
	// for metadata synthesis. Empty if the build is untagged.
	PrimaryRef string
}

// PostStep is a single buildah invocation run after the build.
type PostStep struct {
	Args []string
	Desc string
}

// sink captures the parsed intent of all --push/--load/--output flags.
type sink struct {
	registryPush bool          // push image(s) to a registry
	localStore   bool          // keep image in local containers-storage (the buildah/podman store)
	archives     []archiveSink // image tar exports (docker/oci-archive)
	fsExports    []string      // raw buildah -o specs (filesystem exports: local/tar)
	extraNames   []string      // additional image names requested via output name=
}

type archiveSink struct {
	transport string // "docker-archive" or "oci-archive"
	dest      string
}

// ResolveOutput inspects the output-related options and produces a Plan. It
// returns an error for output requests that cannot be honoured (e.g. push with
// no tag).
func ResolveOutput(o *Options, warn Warnf) (*Plan, error) {
	s, err := parseSinks(o, warn)
	if err != nil {
		return nil, err
	}

	// Default behaviour, matching buildx: with no --push/--load/-o, the image
	// stays in the local store.
	if !s.registryPush && !s.localStore && len(s.archives) == 0 && len(s.fsExports) == 0 {
		s.localStore = true
	}

	tags := append([]string{}, o.Tags...)
	tags = append(tags, s.extraNames...)

	p := &Plan{}
	p.BuildArgs = append(p.BuildArgs, s.fsExports...)

	needsRef := s.registryPush || len(s.archives) > 0
	multi := o.MultiPlatform()

	if multi {
		// Multi-platform builds assemble a manifest list. buildah needs a
		// --manifest name to collect the per-arch images under.
		if len(tags) == 0 {
			if needsRef {
				return nil, fmt.Errorf("multi-platform build requires a tag (-t) when pushing or exporting")
			}
			return nil, fmt.Errorf("multi-platform build requires a tag (-t) so the manifest list can be named")
		}
		p.PrimaryRef = tags[0]
		p.BuildArgs = append(p.BuildArgs, "--manifest", tags[0])
		// Extra tags become additional names for the manifest list.
		for _, t := range tags[1:] {
			p.Post = append(p.Post, PostStep{
				Args: []string{"tag", tags[0], t},
				Desc: "tag manifest list " + t,
			})
		}
		if s.registryPush {
			for _, t := range tags {
				p.Post = append(p.Post, PostStep{
					Args: []string{"manifest", "push", "--all", t, "docker://" + t},
					Desc: "push manifest list " + t,
				})
			}
		}
		for _, ar := range s.archives {
			p.Post = append(p.Post, PostStep{
				Args: []string{"manifest", "push", "--all", tags[0], ar.transport + ":" + ar.dest},
				Desc: "export manifest list to " + ar.transport,
			})
		}
		return p, nil
	}

	// Single-platform: tag the image directly.
	for _, t := range tags {
		p.BuildArgs = append(p.BuildArgs, "--tag", t)
	}
	if len(tags) > 0 {
		p.PrimaryRef = tags[0]
	}

	if s.registryPush {
		if len(tags) == 0 {
			return nil, fmt.Errorf("--push requires a tag (-t)")
		}
		for _, t := range tags {
			p.Post = append(p.Post, PostStep{
				Args: []string{"push", t, "docker://" + t},
				Desc: "push " + t,
			})
		}
	}
	for _, ar := range s.archives {
		ref := dest0(tags)
		args := []string{"push"}
		if ref != "" {
			args = append(args, ref)
		}
		// docker-archive can embed the reference: transport:path:name:tag.
		target := ar.transport + ":" + ar.dest
		if ref != "" {
			target = ar.transport + ":" + ar.dest + ":" + ref
		}
		args = append(args, target)
		p.Post = append(p.Post, PostStep{Args: args, Desc: "export to " + ar.transport})
	}

	return p, nil
}

func dest0(tags []string) string {
	if len(tags) > 0 {
		return tags[0]
	}
	return ""
}

func parseSinks(o *Options, warn Warnf) (sink, error) {
	var s sink
	if o.Push {
		s.registryPush = true
	}
	if o.Load {
		s.localStore = true
	}

	for _, spec := range o.Outputs {
		kv := parseCSV(spec)
		typ := kv["type"]
		if typ == "" {
			// Bare value shorthands: `-` means stdout tar; otherwise a path is a
			// local filesystem export.
			if spec == "-" {
				s.fsExports = append(s.fsExports, "type=tar,dest=-")
			} else {
				s.fsExports = append(s.fsExports, "type=local,dest="+spec)
			}
			continue
		}
		switch typ {
		case "registry":
			s.registryPush = true
			if n := kv["name"]; n != "" {
				s.extraNames = append(s.extraNames, splitNames(n)...)
			}
		case "image":
			if isTrue(kv["push"]) {
				s.registryPush = true
			} else {
				s.localStore = true
			}
			if n := kv["name"]; n != "" {
				s.extraNames = append(s.extraNames, splitNames(n)...)
			}
		case "docker", "oci":
			transport := "docker-archive"
			if typ == "oci" {
				transport = "oci-archive"
			}
			if dest := kv["dest"]; dest != "" {
				s.archives = append(s.archives, archiveSink{transport: transport, dest: dest})
			} else {
				// No dest: load into the local image store (buildx loads into
				// the docker daemon; our store is buildah/podman's).
				s.localStore = true
			}
			if n := kv["name"]; n != "" {
				s.extraNames = append(s.extraNames, splitNames(n)...)
			}
		case "local", "tar":
			s.fsExports = append(s.fsExports, "type="+typ+",dest="+kv["dest"])
		default:
			if warn != nil {
				warn("--output type=%q is not supported by the buildah backend; ignoring", typ)
			}
		}
	}
	return s, nil
}

// parseCSV parses a buildx-style comma-separated key=value list into a map.
// Bare tokens (no '=') are ignored by callers that look up specific keys.
func parseCSV(spec string) map[string]string {
	m := map[string]string{}
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		m[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return m
}

func isTrue(v string) bool {
	switch strings.ToLower(v) {
	case "1", "true", "yes":
		return true
	}
	return false
}

func splitNames(n string) []string {
	var out []string
	for _, p := range strings.Split(n, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
