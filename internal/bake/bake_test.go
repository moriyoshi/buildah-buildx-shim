package bake

import (
	"os"
	"reflect"
	"testing"
)

const sampleHCL = `
variable "TAG" {
  default = "latest"
}
group "default" {
  targets = ["app"]
}
target "base" {
  dockerfile = "Dockerfile"
  args = {
    COMMON = "1"
  }
}
target "app" {
  inherits = ["base"]
  context  = "."
  tags     = ["myapp:${TAG}"]
  args = {
    EXTRA = "2"
  }
}
`

func TestHCLResolveInheritanceAndVars(t *testing.T) {
	cfg, err := parseHCL("docker-bake.hcl", []byte(sampleHCL))
	if err != nil {
		t.Fatal(err)
	}
	targets, err := Resolve(cfg, nil) // default group
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 {
		t.Fatalf("want 1 target, got %d", len(targets))
	}
	app := targets[0]
	if app.Name != "app" {
		t.Errorf("name = %q", app.Name)
	}
	if app.Dockerfile == nil || *app.Dockerfile != "Dockerfile" {
		t.Errorf("inherited dockerfile not applied: %v", app.Dockerfile)
	}
	if !reflect.DeepEqual(app.Tags, []string{"myapp:latest"}) {
		t.Errorf("tags = %v (var interpolation?)", app.Tags)
	}
	if app.Args["COMMON"] != "1" || app.Args["EXTRA"] != "2" {
		t.Errorf("args merge wrong: %v", app.Args)
	}
}

func TestHCLEnvOverridesVariable(t *testing.T) {
	t.Setenv("TAG", "v9")
	cfg, err := parseHCL("docker-bake.hcl", []byte(sampleHCL))
	if err != nil {
		t.Fatal(err)
	}
	targets, err := Resolve(cfg, []string{"app"})
	if err != nil {
		t.Fatal(err)
	}
	if got := targets[0].Tags; !reflect.DeepEqual(got, []string{"myapp:v9"}) {
		t.Errorf("env override not applied: tags = %v", got)
	}
}

func TestApplyOverrides(t *testing.T) {
	tgt := &Target{Name: "app"}
	err := ApplyOverrides([]*Target{tgt}, []string{
		"app.tags=a:1,a:2",
		"*.args.FOO=bar",
		"app.no-cache=true",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(tgt.Tags, []string{"a:1", "a:2"}) {
		t.Errorf("tags = %v", tgt.Tags)
	}
	if tgt.Args["FOO"] != "bar" {
		t.Errorf("args = %v", tgt.Args)
	}
	if tgt.NoCache == nil || !*tgt.NoCache {
		t.Errorf("no-cache not set")
	}
}

func TestComposeParse(t *testing.T) {
	yml := `
services:
  web:
    image: web:latest
    build:
      context: ./web
      args:
        K: V
  db:
    image: postgres
`
	cfg, err := parseCompose([]byte(yml))
	if err != nil {
		t.Fatal(err)
	}
	// Only "web" has a build section.
	if len(cfg.Targets) != 1 {
		t.Fatalf("want 1 target, got %d", len(cfg.Targets))
	}
	web := cfg.Targets[0]
	if web.Context == nil || *web.Context != "./web" {
		t.Errorf("context = %v", web.Context)
	}
	if len(web.Tags) != 1 || web.Tags[0] != "web:latest" {
		t.Errorf("image-as-tag not applied: %v", web.Tags)
	}
}

func TestMain(m *testing.M) {
	// Ensure tests are not affected by an inherited TAG var from the runner.
	os.Unsetenv("TAG")
	os.Exit(m.Run())
}
