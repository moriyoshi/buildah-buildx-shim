package bake

import "encoding/json"

// jsonConfig mirrors the JSON bake file shape, where target/group/variable are
// objects keyed by name rather than repeated blocks.
type jsonConfig struct {
	Variable map[string]struct {
		Default *string `json:"default"`
	} `json:"variable"`
	Group map[string]struct {
		Targets []string `json:"targets"`
	} `json:"group"`
	Target map[string]*Target `json:"target"`
}

// parseJSON parses a JSON bake file. JSON bake values are literal (no HCL-style
// interpolation).
func parseJSON(src []byte) (*Config, error) {
	var jc jsonConfig
	if err := json.Unmarshal(src, &jc); err != nil {
		return nil, err
	}
	cfg := &Config{}
	for name, v := range jc.Variable {
		cfg.Variables = append(cfg.Variables, &Variable{Name: name, Default: v.Default})
	}
	for name, g := range jc.Group {
		cfg.Groups = append(cfg.Groups, &Group{Name: name, Targets: g.Targets})
	}
	for name, t := range jc.Target {
		t.Name = name
		cfg.Targets = append(cfg.Targets, t)
	}
	return cfg, nil
}
