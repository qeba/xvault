package types

import (
	"testing"
)

func TestParseDurationToDays(t *testing.T) {
	tests := []struct {
		name     string
		duration string
		expected int
	}{
		{"empty string", "", 0},
		{"30 days", "30d", 30},
		{"7 days", "7d", 7},
		{"1 day", "1d", 1},
		{"24 hours", "24h", 1},
		{"48 hours", "48h", 2},
		{"25 hours rounds up", "25h", 2},
		{"1 week", "1w", 7},
		{"2 weeks", "2w", 14},
		{"1 month", "1m", 30},
		{"3 months", "3m", 90},
		{"invalid format", "abc", 0},
		{"negative value", "-5d", 0},
		{"zero value", "0d", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseDurationToDays(tt.duration)
			if result != tt.expected {
				t.Errorf("ParseDurationToDays(%q) = %d, want %d", tt.duration, result, tt.expected)
			}
		})
	}
}

func TestRetentionPolicyNormalize(t *testing.T) {
	t.Run("mode all keeps everything", func(t *testing.T) {
		policy := RetentionPolicy{
			Mode:       "all",
			KeepLastN:  intPtr(7),
			MaxAgeDays: intPtr(30),
		}
		policy.Normalize()

		if policy.KeepLastN != nil {
			t.Error("expected KeepLastN to be nil for mode 'all'")
		}
		if policy.MaxAgeDays != nil {
			t.Error("expected MaxAgeDays to be nil for mode 'all'")
		}
	})

	t.Run("mode latest_n uses keep_last_n", func(t *testing.T) {
		policy := RetentionPolicy{
			Mode:      "latest_n",
			KeepLastN: intPtr(7),
		}
		policy.Normalize()

		if policy.KeepLastN == nil || *policy.KeepLastN != 7 {
			t.Errorf("expected KeepLastN to be 7, got %v", policy.KeepLastN)
		}
		if policy.MaxAgeDays != nil {
			t.Error("expected MaxAgeDays to be nil for mode 'latest_n'")
		}
	})

	t.Run("mode within_duration converts to max_age_days", func(t *testing.T) {
		policy := RetentionPolicy{
			Mode:               "within_duration",
			KeepWithinDuration: "30d",
		}
		policy.Normalize()

		if policy.MaxAgeDays == nil {
			t.Fatal("expected MaxAgeDays to be set")
		}
		if *policy.MaxAgeDays != 30 {
			t.Errorf("expected MaxAgeDays to be 30, got %d", *policy.MaxAgeDays)
		}
		if policy.KeepLastN != nil {
			t.Error("expected KeepLastN to be nil for mode 'within_duration'")
		}
	})

	t.Run("within_duration 7d", func(t *testing.T) {
		policy := RetentionPolicy{
			Mode:               "within_duration",
			KeepWithinDuration: "7d",
		}
		policy.Normalize()

		if policy.MaxAgeDays == nil || *policy.MaxAgeDays != 7 {
			t.Errorf("expected MaxAgeDays to be 7, got %v", policy.MaxAgeDays)
		}
	})

	t.Run("within_duration 1w", func(t *testing.T) {
		policy := RetentionPolicy{
			Mode:               "within_duration",
			KeepWithinDuration: "1w",
		}
		policy.Normalize()

		if policy.MaxAgeDays == nil || *policy.MaxAgeDays != 7 {
			t.Errorf("expected MaxAgeDays to be 7, got %v", policy.MaxAgeDays)
		}
	})

	t.Run("fallback: keep_within_duration without mode", func(t *testing.T) {
		policy := RetentionPolicy{
			KeepWithinDuration: "14d",
		}
		policy.Normalize()

		if policy.MaxAgeDays == nil || *policy.MaxAgeDays != 14 {
			t.Errorf("expected MaxAgeDays to be 14, got %v", policy.MaxAgeDays)
		}
	})
}

func intPtr(i int) *int {
	return &i
}
