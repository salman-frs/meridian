package runtime

import "testing"

func TestParseRunningState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{
			name:   "plain true",
			output: "true\n",
			want:   true,
		},
		{
			name: "warning then true",
			output: "time=\"2026-03-13T18:30:25+01:00\" level=warning msg=\"failed to inspect NetNS\"\ntrue\n",
			want:   true,
		},
		{
			name:   "plain false",
			output: "false\n",
			want:   false,
		},
		{
			name:   "unexpected output",
			output: "warning only\n",
			want:   false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := parseRunningState([]byte(tt.output)); got != tt.want {
				t.Fatalf("parseRunningState() = %v, want %v", got, tt.want)
			}
		})
	}
}
