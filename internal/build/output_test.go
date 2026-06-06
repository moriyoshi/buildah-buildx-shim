package build

import (
	"reflect"
	"testing"
)

func TestResolveOutput(t *testing.T) {
	tests := []struct {
		name      string
		o         Options
		wantBuild []string
		wantPost  [][]string
		wantRef   string
		wantErr   bool
	}{
		{
			name:      "default loads to store",
			o:         Options{Tags: []string{"img:latest"}},
			wantBuild: []string{"--tag", "img:latest"},
			wantRef:   "img:latest",
		},
		{
			name:      "push single platform",
			o:         Options{Tags: []string{"reg/img:1"}, Push: true},
			wantBuild: []string{"--tag", "reg/img:1"},
			wantPost:  [][]string{{"push", "reg/img:1", "docker://reg/img:1"}},
			wantRef:   "reg/img:1",
		},
		{
			name:      "push multi platform uses manifest",
			o:         Options{Tags: []string{"reg/img:1"}, Push: true, Platforms: []string{"linux/amd64,linux/arm64"}},
			wantBuild: []string{"--manifest", "reg/img:1"},
			wantPost:  [][]string{{"manifest", "push", "--all", "reg/img:1", "docker://reg/img:1"}},
			wantRef:   "reg/img:1",
		},
		{
			name:      "output docker archive",
			o:         Options{Tags: []string{"img:1"}, Outputs: []string{"type=docker,dest=out.tar"}},
			wantBuild: []string{"--tag", "img:1"},
			wantPost:  [][]string{{"push", "img:1", "docker-archive:out.tar:img:1"}},
			wantRef:   "img:1",
		},
		{
			name:      "output local fs export",
			o:         Options{Outputs: []string{"type=local,dest=./out"}},
			wantBuild: []string{"type=local,dest=./out"},
		},
		{
			name:    "push without tag errors",
			o:       Options{Push: true},
			wantErr: true,
		},
		{
			name:    "multi platform without tag errors",
			o:       Options{Platforms: []string{"linux/amd64,linux/arm64"}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := ResolveOutput(&tt.o, nil)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(p.BuildArgs, tt.wantBuild) {
				t.Errorf("BuildArgs = %v, want %v", p.BuildArgs, tt.wantBuild)
			}
			var gotPost [][]string
			for _, s := range p.Post {
				gotPost = append(gotPost, s.Args)
			}
			if !reflect.DeepEqual(gotPost, tt.wantPost) {
				t.Errorf("Post = %v, want %v", gotPost, tt.wantPost)
			}
			if p.PrimaryRef != tt.wantRef {
				t.Errorf("PrimaryRef = %q, want %q", p.PrimaryRef, tt.wantRef)
			}
		})
	}
}
