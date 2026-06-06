// Package bake implements a practical subset of `docker buildx bake`: it reads
// HCL, JSON, and compose bake files, resolves the requested targets (with
// group expansion, inheritance and variable interpolation), and exposes them as
// build invocations.
//
// Out of scope for this implementation: matrix builds and the full bake
// function library. Such constructs are reported and skipped.
package bake

import "github.com/hashicorp/hcl/v2"

// Target is a single bake target. Pointer/slice/map fields distinguish "unset"
// from "set to zero", which matters for inheritance overrides.
type Target struct {
	Name string `hcl:"name,label" json:"-"`

	Context    *string           `hcl:"context,optional" json:"context,omitempty"`
	Dockerfile *string           `hcl:"dockerfile,optional" json:"dockerfile,omitempty"`
	Target     *string           `hcl:"target,optional" json:"target,omitempty"`
	Tags       []string          `hcl:"tags,optional" json:"tags,omitempty"`
	Args       map[string]string `hcl:"args,optional" json:"args,omitempty"`
	Labels     map[string]string `hcl:"labels,optional" json:"labels,omitempty"`
	Platforms  []string          `hcl:"platforms,optional" json:"platforms,omitempty"`
	Secret     []string          `hcl:"secret,optional" json:"secret,omitempty"`
	SSH        []string          `hcl:"ssh,optional" json:"ssh,omitempty"`
	NoCache    *bool             `hcl:"no-cache,optional" json:"no-cache,omitempty"`
	Pull       *bool             `hcl:"pull,optional" json:"pull,omitempty"`
	CacheFrom  []string          `hcl:"cache-from,optional" json:"cache-from,omitempty"`
	CacheTo    []string          `hcl:"cache-to,optional" json:"cache-to,omitempty"`
	Output     []string          `hcl:"output,optional" json:"output,omitempty"`
	Inherits   []string          `hcl:"inherits,optional" json:"inherits,omitempty"`
}

// Group is a named collection of targets (and/or other groups).
type Group struct {
	Name    string   `hcl:"name,label" json:"-"`
	Targets []string `hcl:"targets,optional" json:"targets,omitempty"`
}

// Variable is a bake variable with an optional default; overridable via an
// environment variable of the same name.
type Variable struct {
	Name    string  `hcl:"name,label" json:"-"`
	Default *string `hcl:"default,optional" json:"default,omitempty"`
}

// Config is the merged result of all parsed bake files.
type Config struct {
	Variables []*Variable `hcl:"variable,block" json:"-"`
	Groups    []*Group    `hcl:"group,block" json:"-"`
	Targets   []*Target   `hcl:"target,block" json:"-"`
	// Remain absorbs unsupported HCL blocks (e.g. function) so decoding does
	// not fail on the common subset we don't implement.
	Remain hcl.Body `hcl:",remain" json:"-"`
}

func (c *Config) target(name string) *Target {
	for _, t := range c.Targets {
		if t.Name == name {
			return t
		}
	}
	return nil
}

func (c *Config) group(name string) *Group {
	for _, g := range c.Groups {
		if g.Name == name {
			return g
		}
	}
	return nil
}
