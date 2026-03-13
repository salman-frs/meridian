package app

import (
	"errors"
	"testing"

	"github.com/salman-frs/meridian/internal/model"
)

func TestShouldRetryRuntimeRun(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "retries port allocation failure",
			err:  &model.ExitError{Code: 3, Err: errors.New("failed to start collector container: port is already allocated")},
			want: true,
		},
		{
			name: "retries address in use failure",
			err:  &model.ExitError{Code: 3, Err: errors.New("listen tcp 127.0.0.1:4317: bind: address already in use")},
			want: true,
		},
		{
			name: "does not retry validation failure",
			err:  &model.ExitError{Code: 2, Err: errors.New("validation failed")},
			want: false,
		},
		{
			name: "does not retry unrelated runtime failure",
			err:  &model.ExitError{Code: 3, Err: errors.New("collector exited before startup timeout")},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldRetryRuntimeRun(tt.err); got != tt.want {
				t.Fatalf("shouldRetryRuntimeRun() = %v, want %v", got, tt.want)
			}
		})
	}
}
