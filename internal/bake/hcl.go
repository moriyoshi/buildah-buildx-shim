package bake

import (
	"fmt"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"
)

// parseHCL parses a single HCL bake file. Variable blocks are read first to
// build an evaluation context (env overrides defaults), then the rest of the
// document is decoded with variable interpolation enabled.
func parseHCL(filename string, src []byte) (*Config, error) {
	p := hclparse.NewParser()
	f, diags := p.ParseHCL(src, filename)
	if diags.HasErrors() {
		return nil, diagErr(filename, diags)
	}

	// Pass 1: variables only, evaluated against an empty context (literals).
	var vars struct {
		Variables []*Variable `hcl:"variable,block"`
		Remain    hcl.Body    `hcl:",remain"`
	}
	if d := gohcl.DecodeBody(f.Body, nil, &vars); d.HasErrors() {
		return nil, diagErr(filename, d)
	}

	ctx := &hcl.EvalContext{Variables: map[string]cty.Value{}}
	for _, v := range vars.Variables {
		val := ""
		if env, ok := os.LookupEnv(v.Name); ok {
			val = env
		} else if v.Default != nil {
			val = *v.Default
		}
		ctx.Variables[v.Name] = cty.StringVal(val)
	}

	// Pass 2: full document with interpolation.
	var cfg Config
	if d := gohcl.DecodeBody(f.Body, ctx, &cfg); d.HasErrors() {
		return nil, diagErr(filename, d)
	}
	return &cfg, nil
}

func diagErr(filename string, diags hcl.Diagnostics) error {
	return fmt.Errorf("parsing %s: %s", filename, diags.Error())
}
