package preflight

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseCleanSyncMaxAge(t *testing.T) {
	defaultMaxAge := 30 * 24 * time.Hour

	tests := []struct {
		name  string
		value string
		want  time.Duration
	}{
		{name: "unset uses default", value: "", want: defaultMaxAge},
		{name: "custom days", value: "7", want: 7 * 24 * time.Hour},
		{name: "zero uses default", value: "0", want: defaultMaxAge},
		{name: "negative uses default", value: "-5", want: defaultMaxAge},
		{name: "non numeric uses default", value: "thirty", want: defaultMaxAge},
		{name: "duration string uses default", value: "720h", want: defaultMaxAge},
		{name: "out of range uses default", value: "100000", want: defaultMaxAge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parseCleanSyncMaxAge(tt.value))
		})
	}
}

func TestDefaultCleanSyncMaxAgeIsThirtyDays(t *testing.T) {
	assert.Equal(t, 30*24*time.Hour, cleanSyncMaxAgeFromDays(defaultCleanSyncMaxAgeDays))
}
