package bake

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// looksLikeJSON reports whether src appears to be a JSON document (its first
// non-whitespace byte is '{' or '[').
func looksLikeJSON(src []byte) bool {
	for _, b := range src {
		switch b {
		case ' ', '\t', '\r', '\n':
			continue
		case '{', '[':
			return true
		default:
			return false
		}
	}
	return false
}

// defaultFiles is the search order used when no -f is given. Only files that
// exist are loaded.
var defaultFiles = []string{
	"docker-bake.hcl",
	"docker-bake.override.hcl",
	"docker-bake.json",
	"docker-compose.yml",
	"docker-compose.yaml",
	"compose.yaml",
	"compose.yml",
}

// Load reads and merges the given bake files (or the default set if none are
// given). Later files override earlier ones for same-named groups/targets.
func Load(files []string) (*Config, error) {
	if len(files) == 0 {
		for _, f := range defaultFiles {
			if _, err := os.Stat(f); err == nil {
				files = append(files, f)
			}
		}
		if len(files) == 0 {
			return nil, fmt.Errorf("no bake definition file found (looked for %s)", strings.Join(defaultFiles, ", "))
		}
	}

	merged := &Config{}
	for _, f := range files {
		src, err := readSource(f)
		if err != nil {
			return nil, err
		}
		cfg, err := parseFile(f, src)
		if err != nil {
			return nil, err
		}
		merge(merged, cfg)
	}
	return merged, nil
}

// readSource reads a bake definition from a file, or from stdin when the name
// is "-" (used by `docker compose build`, which pipes the generated definition).
func readSource(name string) ([]byte, error) {
	if name == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(name)
}

func parseFile(filename string, src []byte) (*Config, error) {
	// stdin has no extension; detect by content (compose emits JSON or HCL).
	if filename == "-" {
		if looksLikeJSON(src) {
			return parseJSON(src)
		}
		return parseHCL("(stdin)", src)
	}
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".json":
		return parseJSON(src)
	case ".yml", ".yaml":
		return parseCompose(src)
	case ".hcl":
		return parseHCL(filename, src)
	default:
		// Fall back to compose for compose-named files, else HCL.
		base := strings.ToLower(filepath.Base(filename))
		if strings.HasPrefix(base, "docker-compose") || strings.HasPrefix(base, "compose") {
			return parseCompose(src)
		}
		return parseHCL(filename, src)
	}
}

// merge folds src into dst, replacing same-named groups and targets and unioning
// variables (later wins).
func merge(dst, src *Config) {
	for _, v := range src.Variables {
		if existing := findVar(dst.Variables, v.Name); existing != nil {
			*existing = *v
		} else {
			dst.Variables = append(dst.Variables, v)
		}
	}
	for _, g := range src.Groups {
		if existing := dst.group(g.Name); existing != nil {
			*existing = *g
		} else {
			dst.Groups = append(dst.Groups, g)
		}
	}
	for _, t := range src.Targets {
		if existing := dst.target(t.Name); existing != nil {
			*existing = *mergeTarget(existing, t)
		} else {
			dst.Targets = append(dst.Targets, t)
		}
	}
}

func findVar(vars []*Variable, name string) *Variable {
	for _, v := range vars {
		if v.Name == name {
			return v
		}
	}
	return nil
}
