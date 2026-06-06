package build

import (
	"reflect"
	"testing"
)

func TestCommonArgs(t *testing.T) {
	tests := []struct {
		name string
		o    Options
		want []string
	}{
		{
			name: "passthrough basics",
			o: Options{
				File:      "Containerfile",
				BuildArgs: []string{"A=1", "B=2"},
				Target:    "prod",
				Platforms: []string{"linux/amd64"},
				NoCache:   true,
				Labels:    []string{"k=v"},
				Network:   "host",
			},
			want: []string{
				"--file", "Containerfile",
				"--build-arg", "A=1", "--build-arg", "B=2",
				"--target", "prod",
				"--platform", "linux/amd64",
				"--no-cache",
				"--label", "k=v",
				"--network", "host",
			},
		},
		{
			name: "pull maps to pull=always",
			o:    Options{Pull: true},
			want: []string{"--pull=always"},
		},
		{
			name: "cache registry ref extraction",
			o: Options{
				CacheFrom: []string{"type=registry,ref=example.com/cache"},
				CacheTo:   []string{"example.com/cache2"},
			},
			want: []string{
				"--cache-from", "example.com/cache",
				"--cache-to", "example.com/cache2",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CommonArgs(&tt.o, nil)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CommonArgs() =\n  %v\nwant\n  %v", got, tt.want)
			}
		})
	}
}

func TestTranslateSSH(t *testing.T) {
	tests := []struct {
		in   []string
		want []string
	}{
		{[]string{"default="}, []string{"default"}},          // compose's form
		{[]string{"default"}, []string{"default"}},           // already bare
		{[]string{"id=/run/sock"}, []string{"id=/run/sock"}}, // explicit socket kept
		{[]string{"key=/k1,/k2"}, []string{"key=/k1,/k2"}},   // key list kept
	}
	for _, tt := range tests {
		if got := translateSSH(tt.in); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("translateSSH(%v) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestCommonArgsDropsUnsupportedCache(t *testing.T) {
	o := Options{CacheFrom: []string{"type=gha"}}
	if got := CommonArgs(&o, nil); len(got) != 0 {
		t.Errorf("expected gha cache to be dropped, got %v", got)
	}
}
