package configio

import "testing"

func FuzzInterpolateValue(f *testing.F) {
	f.Add("endpoint: ${OTLP_ENDPOINT}")
	f.Add("api_key: ${env:API_KEY}")
	f.Add("mixed ${A} ${B:-fallback} ${ C }")

	f.Fuzz(func(t *testing.T, input string) {
		_, refs, missing := InterpolateValue(input, map[string]string{
			"OTLP_ENDPOINT": "localhost:4317",
			"API_KEY":       "secret",
			"A":             "1",
		})

		if len(missing) > len(refs) {
			t.Fatalf("missing refs %d cannot exceed references %d", len(missing), len(refs))
		}
	})
}
