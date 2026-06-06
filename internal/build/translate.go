package build

import "strings"

// Warnf reports a non-fatal compatibility message.
type Warnf func(format string, a ...any)

// CommonArgs translates the buildx flags that map onto `buildah build` and do
// not participate in output/tag/metadata handling (those are resolved
// separately by ResolveOutput and the caller). The returned slice does not
// include the leading "build" verb, the tags, the iidfile, or the positional
// context.
func CommonArgs(o *Options, warn Warnf) []string {
	var a []string

	add := func(flag string, vals []string) {
		for _, v := range vals {
			a = append(a, flag, v)
		}
	}

	if o.File != "" {
		a = append(a, "--file", o.File)
	}
	add("--build-arg", o.BuildArgs)
	add("--build-context", o.BuildContexts)
	if o.Target != "" {
		a = append(a, "--target", o.Target)
	}
	for _, p := range o.Platforms {
		a = append(a, "--platform", p)
	}
	if o.NoCache {
		a = append(a, "--no-cache")
	}
	if o.Pull {
		// buildx --pull forces a pull of newer base images; the buildah
		// equivalent is --pull=always.
		a = append(a, "--pull=always")
	}
	add("--label", o.Labels)
	add("--annotation", o.Annotations)
	if o.Network != "" {
		a = append(a, "--network", o.Network)
	}
	add("--add-host", o.AddHosts)
	add("--cache-from", translateCacheRefs(o.CacheFrom, "cache-from", warn))
	add("--cache-to", translateCacheRefs(o.CacheTo, "cache-to", warn))
	if o.CgroupParent != "" {
		a = append(a, "--cgroup-parent", o.CgroupParent)
	}
	if o.ShmSize != "" {
		a = append(a, "--shm-size", o.ShmSize)
	}
	add("--ulimit", o.Ulimits)
	add("--secret", o.Secrets)
	add("--ssh", translateSSH(o.SSH))
	if o.Quiet {
		a = append(a, "--quiet")
	}

	warnUnsupported(o, warn)
	return a
}

// translateSSH normalizes buildx --ssh entries for buildah. docker compose
// emits `default=` (an id with an empty socket value), which buildx reads as
// "use the agent from $SSH_AUTH_SOCK" but buildah takes literally and then
// fails with `stat : no such file or directory`. Dropping the trailing empty
// `=value` restores the agent-socket behaviour.
func translateSSH(specs []string) []string {
	if len(specs) == 0 {
		return nil
	}
	out := make([]string, 0, len(specs))
	for _, s := range specs {
		if id, val, ok := strings.Cut(s, "="); ok && val == "" {
			out = append(out, id)
		} else {
			out = append(out, s)
		}
	}
	return out
}

// translateCacheRefs converts buildx cache specs into the bare repository
// references buildah expects. buildx accepts both `ref` (a bare image
// reference) and the structured `type=registry,ref=...` form; buildah only
// understands the bare reference. Non-registry cache backends are dropped.
func translateCacheRefs(specs []string, flag string, warn Warnf) []string {
	var out []string
	for _, s := range specs {
		if !containsKV(s) {
			out = append(out, s) // already a bare ref
			continue
		}
		kv := parseCSV(s)
		typ := kv["type"]
		if typ != "" && typ != "registry" {
			if warn != nil {
				warn("--%s type=%q is not supported by the buildah backend; ignoring", flag, typ)
			}
			continue
		}
		if ref := kv["ref"]; ref != "" {
			out = append(out, ref)
		} else if warn != nil {
			warn("--%s %q has no ref=; ignoring", flag, s)
		}
	}
	return out
}

// warnUnsupported emits warnings for buildx flags that have no buildah analogue.
func warnUnsupported(o *Options, warn Warnf) {
	if warn == nil {
		return
	}
	type u struct {
		set  bool
		name string
	}
	for _, f := range []u{
		{len(o.Allow) > 0, "--allow"},
		{len(o.Attests) > 0, "--attest"},
		{len(o.NoCacheFilter) > 0, "--no-cache-filter"},
		{o.Provenance != "", "--provenance"},
		{o.SBOM != "", "--sbom"},
		{o.Builder != "", "--builder"},
	} {
		if f.set {
			warn("%s is not supported by the buildah backend; ignoring", f.name)
		}
	}
	if o.Progress != "" && o.Progress != "auto" {
		// Progress style is purely cosmetic; note it once but stay quiet for auto.
		warn("--progress=%s has no buildah equivalent; using buildah's default output", o.Progress)
	}
}

// containsKV reports whether s uses the structured key=value[,key=value] form
// rather than a bare image reference.
func containsKV(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return true
		}
	}
	return false
}
