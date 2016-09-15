package rotatelogs

import (
	"fmt"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/assert"
)

func TestGenFilename(t *testing.T) {
	// Mock time
	ts := []time.Time{
		time.Time{},
		(time.Time{}).Add(24 * time.Hour),
	}

	for _, xt := range ts {
		rl := New(
			"/path/to/%Y/%m/%d",
			WithClock(clockwork.NewFakeClockAt(xt)),
		)
		defer rl.Close()

		fn, err := rl.genFilename()
		if !assert.NoError(t, err, "filename generation should succeed") {
			return
		}

		expected := fmt.Sprintf("/path/to/%04d/%02d/%02d",
			xt.Year(),
			xt.Month(),
			xt.Day(),
		)

		if !assert.Equal(t, expected, fn) {
			return
		}
	}
}
