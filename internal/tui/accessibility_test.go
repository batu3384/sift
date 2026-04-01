package tui

import "testing"

func TestTruthyEnv(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"1", "true", "TRUE", " yes ", "on"} {
		if !truthyEnv(value) {
			t.Fatalf("expected %q to enable env flag", value)
		}
	}
	for _, value := range []string{"", "0", "false", "no", "off"} {
		if truthyEnv(value) {
			t.Fatalf("expected %q to keep env flag disabled", value)
		}
	}
}
