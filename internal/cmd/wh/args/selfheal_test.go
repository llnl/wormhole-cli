package args

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseSelfHeal(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		got, err := ParseSelfHeal(nil, "test")
		assert.NoError(t, err)
		assert.Equal(t, DefaultSelfHealArgs(), got)
	})

	t.Run("overrides", func(t *testing.T) {
		got, err := ParseSelfHeal([]byte(`[self-heal]
min-retry-backoff = "250ms"
max-retry-backoff = "9s"
`), "test")
		assert.NoError(t, err)
		assert.Equal(t, 250*time.Millisecond, got.MinRetryBackoff)
		assert.Equal(t, 9*time.Second, got.MaxRetryBackoff)
	})

	for name, content := range map[string]string{
		"negative backoff": `[self-heal]
min-retry-backoff = "-1s"`,
		"minimum over maximum": `[self-heal]
min-retry-backoff = "2s"
max-retry-backoff = "1s"`,
		"invalid duration": `[self-heal]
max-retry-backoff = "later"`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := ParseSelfHeal([]byte(content), "test")
			assert.Error(t, err)
		})
	}
}
