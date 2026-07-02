package app

import "testing"

func TestNormalizeAITaskStatus(t *testing.T) {
	tests := map[string]string{
		"pending":         "pending",
		"processing":      "processing",
		"succeeded":       "succeeded",
		"completed":       "succeeded",
		"failed":          "failed",
		"config_required": "config_required",
	}
	for input, want := range tests {
		got, err := normalizeAITaskStatus(input)
		if err != nil {
			t.Fatalf("expected %s to be supported: %v", input, err)
		}
		if got != want {
			t.Fatalf("expected %s to normalize to %s, got %s", input, want, got)
		}
	}
	if _, err := normalizeAITaskStatus("unknown"); err == nil {
		t.Fatal("unknown status should be rejected")
	}
}
