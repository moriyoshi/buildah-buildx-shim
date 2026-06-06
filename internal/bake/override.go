package bake

import (
	"fmt"
	"strings"
)

// ApplyOverrides applies buildx `--set` overrides to resolved targets. Each
// override is `PATTERN.key[.subkey]=value`, where PATTERN is a target name or
// `*` for all. Unknown keys are reported via the returned error.
func ApplyOverrides(targets []*Target, sets []string) error {
	for _, s := range sets {
		lhs, value, ok := strings.Cut(s, "=")
		if !ok {
			return fmt.Errorf("invalid --set %q: expected PATTERN.key=value", s)
		}
		pattern, key, ok := strings.Cut(lhs, ".")
		if !ok {
			return fmt.Errorf("invalid --set %q: expected PATTERN.key=value", s)
		}
		for _, t := range targets {
			if pattern != "*" && pattern != t.Name {
				continue
			}
			if err := applyOne(t, key, value); err != nil {
				return err
			}
		}
	}
	return nil
}

func applyOne(t *Target, key, value string) error {
	switch {
	case key == "context":
		t.Context = &value
	case key == "dockerfile":
		t.Dockerfile = &value
	case key == "target":
		t.Target = &value
	case key == "tags":
		t.Tags = splitList(value)
	case key == "platform" || key == "platforms":
		t.Platforms = splitList(value)
	case key == "output":
		t.Output = splitList(value)
	case key == "secret":
		t.Secret = splitList(value)
	case key == "ssh":
		t.SSH = splitList(value)
	case key == "cache-from":
		t.CacheFrom = splitList(value)
	case key == "cache-to":
		t.CacheTo = splitList(value)
	case key == "no-cache":
		b := isTruthy(value)
		t.NoCache = &b
	case key == "pull":
		b := isTruthy(value)
		t.Pull = &b
	case strings.HasPrefix(key, "args."):
		if t.Args == nil {
			t.Args = map[string]string{}
		}
		t.Args[strings.TrimPrefix(key, "args.")] = value
	case strings.HasPrefix(key, "labels."):
		if t.Labels == nil {
			t.Labels = map[string]string{}
		}
		t.Labels[strings.TrimPrefix(key, "labels.")] = value
	default:
		return fmt.Errorf("unsupported --set key %q", key)
	}
	return nil
}

func splitList(v string) []string {
	var out []string
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func isTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes":
		return true
	}
	return false
}
