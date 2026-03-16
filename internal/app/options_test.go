package app

import (
	"testing"
	"time"
)

func TestValidateRuntimeOptions(t *testing.T) {
	t.Parallel()

	global := &GlobalOptions{ConfigPath: "collector.yaml", Format: "human"}
	base := newRuntimeOptions()

	tests := []struct {
		name    string
		mutate  func(*RuntimeOptions)
		wantErr bool
	}{
		{
			name: "accepts defaults",
			mutate: func(*RuntimeOptions) {
			},
		},
		{
			name: "rejects invalid mode",
			mutate: func(opts *RuntimeOptions) {
				opts.Mode = "broken"
			},
			wantErr: true,
		},
		{
			name: "rejects invalid render graph",
			mutate: func(opts *RuntimeOptions) {
				opts.RenderGraph = "pdf"
			},
			wantErr: true,
		},
		{
			name: "requires diff inputs when changed only",
			mutate: func(opts *RuntimeOptions) {
				opts.ChangedOnly = true
			},
			wantErr: true,
		},
		{
			name: "rejects non-positive timeout",
			mutate: func(opts *RuntimeOptions) {
				opts.Timeout = 0
			},
			wantErr: true,
		},
		{
			name: "accepts changed only with base ref",
			mutate: func(opts *RuntimeOptions) {
				opts.ChangedOnly = true
				opts.Diff.BaseRef = "main"
				opts.Diff.HeadRef = "HEAD"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts := *base
			opts.Timeout = 5 * time.Second
			tt.mutate(&opts)
			err := validateRuntimeOptions(global, &opts)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateRuntimeOptions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResolveRuntimeOptionsReturnsTypedValues(t *testing.T) {
	t.Parallel()

	global := &GlobalOptions{ConfigPath: "collector.yaml", Format: "json"}
	opts := newRuntimeOptions()
	opts.Engine = "docker"
	opts.Mode = "tee"
	opts.RenderGraph = "svg"

	resolved, err := resolveRuntimeOptions(global, opts)
	if err != nil {
		t.Fatalf("resolveRuntimeOptions() error = %v", err)
	}
	if resolved.Engine != "docker" || resolved.Mode != "tee" || resolved.RenderGraph != graphRenderSVG {
		t.Fatalf("resolveRuntimeOptions() = %#v", resolved)
	}
}
