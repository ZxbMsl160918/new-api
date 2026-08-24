package model_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetModelContextLimit(t *testing.T) {
	original := GetContextLimitSettings().ModelContextLimits
	t.Cleanup(func() { GetContextLimitSettings().ModelContextLimits = original })

	GetContextLimitSettings().ModelContextLimits = map[string]int{
		"deepseek-v4-pro": 300000,
		"gpt-5.6-luna*":   200000,
		"gpt-5.6*":        128000,
		"gpt-5.6":         100000,
		"expired-model":   0,
	}

	tests := []struct {
		name      string
		model     string
		wantLimit int
		wantFound bool
	}{
		{"exact match", "deepseek-v4-pro", 300000, true},
		{"unconfigured model", "claude-opus-4-8", 0, false},
		{"prefix wildcard match", "gpt-5.6-luna-chat", 200000, true},
		{"longest prefix wins", "gpt-5.6-luna-thinking", 200000, true},
		{"shorter prefix fallback", "gpt-5.6-mini", 128000, true},
		{"prefix must not overmatch", "gpt-5.7-luna", 0, false},
		{"exact beats wildcard", "gpt-5.6", 100000, true},
		{"non-positive limit ignored", "expired-model", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limit, found := GetModelContextLimit(tt.model)
			assert.Equal(t, tt.wantFound, found)
			assert.Equal(t, tt.wantLimit, limit)
		})
	}
}

func TestValidateModelContextLimits(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty map ok", `{}`, false},
		{"exact model ok", `{"deepseek-v4-pro": 300000}`, false},
		{"trailing wildcard ok", `{"gpt-5.6-luna*": 200000}`, false},
		{"inner wildcard rejected", `{"gpt-*5": 100}`, true},
		{"multiple wildcards rejected", `{"gpt-5.6**": 100}`, true},
		{"bare wildcard rejected", `{"*": 100}`, true},
		{"empty model rejected", `{"": 100}`, true},
		{"non-positive limit rejected", `{"gpt-5": 0}`, true},
		{"invalid json rejected", `{gpt}`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateModelContextLimits(tt.value)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
