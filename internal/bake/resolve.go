package bake

// Resolve expands the requested names (targets or groups) into a concrete,
// de-duplicated, inheritance-applied list of targets. When names is empty, the
// "default" group is used if present, otherwise every target.
func Resolve(cfg *Config, names []string) ([]*Target, error) {
	if len(names) == 0 {
		if g := cfg.group("default"); g != nil {
			names = g.Targets
		} else {
			for _, t := range cfg.Targets {
				names = append(names, t.Name)
			}
		}
	}

	var order []string
	seen := map[string]bool{}
	var expand func(name string) error
	expand = func(name string) error {
		if g := cfg.group(name); g != nil {
			for _, sub := range g.Targets {
				if err := expand(sub); err != nil {
					return err
				}
			}
			return nil
		}
		if cfg.target(name) == nil {
			return &UnknownTargetError{Name: name}
		}
		if !seen[name] {
			seen[name] = true
			order = append(order, name)
		}
		return nil
	}
	for _, n := range names {
		if err := expand(n); err != nil {
			return nil, err
		}
	}

	out := make([]*Target, 0, len(order))
	for _, name := range order {
		t, err := resolveTarget(cfg, name, map[string]bool{})
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

// resolveTarget returns a target with its inheritance chain applied. visiting
// guards against inheritance cycles.
func resolveTarget(cfg *Config, name string, visiting map[string]bool) (*Target, error) {
	base := cfg.target(name)
	if base == nil {
		return nil, &UnknownTargetError{Name: name}
	}
	if visiting[name] {
		return nil, &InheritCycleError{Name: name}
	}
	visiting[name] = true
	defer delete(visiting, name)

	acc := &Target{Name: name}
	for _, parentName := range base.Inherits {
		parent, err := resolveTarget(cfg, parentName, visiting)
		if err != nil {
			return nil, err
		}
		acc = mergeTarget(acc, parent)
	}
	acc = mergeTarget(acc, base)
	acc.Name = name
	acc.Inherits = nil
	return acc, nil
}

// mergeTarget returns a new target where set fields in override win over base.
func mergeTarget(base, override *Target) *Target {
	out := *base
	if override.Context != nil {
		out.Context = override.Context
	}
	if override.Dockerfile != nil {
		out.Dockerfile = override.Dockerfile
	}
	if override.Target != nil {
		out.Target = override.Target
	}
	if override.NoCache != nil {
		out.NoCache = override.NoCache
	}
	if override.Pull != nil {
		out.Pull = override.Pull
	}
	if len(override.Tags) > 0 {
		out.Tags = override.Tags
	}
	if len(override.Platforms) > 0 {
		out.Platforms = override.Platforms
	}
	if len(override.Secret) > 0 {
		out.Secret = override.Secret
	}
	if len(override.SSH) > 0 {
		out.SSH = override.SSH
	}
	if len(override.CacheFrom) > 0 {
		out.CacheFrom = override.CacheFrom
	}
	if len(override.CacheTo) > 0 {
		out.CacheTo = override.CacheTo
	}
	if len(override.Output) > 0 {
		out.Output = override.Output
	}
	out.Args = mergeMap(base.Args, override.Args)
	out.Labels = mergeMap(base.Labels, override.Labels)
	return &out
}

func mergeMap(base, override map[string]string) map[string]string {
	if base == nil && override == nil {
		return nil
	}
	m := map[string]string{}
	for k, v := range base {
		m[k] = v
	}
	for k, v := range override {
		m[k] = v
	}
	return m
}

// UnknownTargetError indicates a requested target/group does not exist.
type UnknownTargetError struct{ Name string }

func (e *UnknownTargetError) Error() string { return "unknown target or group: " + e.Name }

// InheritCycleError indicates an inheritance cycle.
type InheritCycleError struct{ Name string }

func (e *InheritCycleError) Error() string { return "inheritance cycle through target: " + e.Name }
