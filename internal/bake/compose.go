package bake

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// parseCompose extracts build targets from a compose file. Each service with a
// `build:` section becomes a target named after the service. The service
// `image:` is used as a tag when the build section doesn't specify tags.
func parseCompose(src []byte) (*Config, error) {
	var doc struct {
		Services map[string]struct {
			Image string    `yaml:"image"`
			Build yaml.Node `yaml:"build"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal(src, &doc); err != nil {
		return nil, err
	}

	cfg := &Config{}
	var names []string
	for name, svc := range doc.Services {
		if svc.Build.IsZero() {
			continue
		}
		t := &Target{Name: name}
		if err := applyComposeBuild(t, svc.Build); err != nil {
			return nil, fmt.Errorf("service %q: %w", name, err)
		}
		if len(t.Tags) == 0 && svc.Image != "" {
			t.Tags = []string{svc.Image}
		}
		cfg.Targets = append(cfg.Targets, t)
		names = append(names, name)
	}
	if len(names) > 0 {
		cfg.Groups = append(cfg.Groups, &Group{Name: "default", Targets: names})
	}
	return cfg, nil
}

// applyComposeBuild fills t from a compose `build:` node, which may be a bare
// context string or a mapping.
func applyComposeBuild(t *Target, node yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		ctx := node.Value
		t.Context = &ctx
		return nil
	}
	var b struct {
		Context    string            `yaml:"context"`
		Dockerfile string            `yaml:"dockerfile"`
		Target     string            `yaml:"target"`
		Args       map[string]string `yaml:"args"`
		Labels     map[string]string `yaml:"labels"`
		Tags       []string          `yaml:"tags"`
		Platforms  []string          `yaml:"platforms"`
		CacheFrom  []string          `yaml:"cache_from"`
		CacheTo    []string          `yaml:"cache_to"`
	}
	if err := node.Decode(&b); err != nil {
		return err
	}
	if b.Context != "" {
		t.Context = &b.Context
	}
	if b.Dockerfile != "" {
		t.Dockerfile = &b.Dockerfile
	}
	if b.Target != "" {
		t.Target = &b.Target
	}
	t.Args = b.Args
	t.Labels = b.Labels
	t.Tags = b.Tags
	t.Platforms = b.Platforms
	t.CacheFrom = b.CacheFrom
	t.CacheTo = b.CacheTo
	return nil
}
