package fileutil_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/lestrrat-go/file-rotatelogs/internal/fileutil"
	"github.com/lestrrat-go/strftime"
	"github.com/stretchr/testify/assert"
)

func TestGenerateFn(t *testing.T) {
	// Mock time
	ts := []time.Time{
		{},
		(time.Time{}).Add(24 * time.Hour),
	}

	for _, xt := range ts {
		pattern, err := strftime.New("/path/to/%Y/%m/%d")
		if !assert.NoError(t, err, `strftime.New should succeed`) {
			return
		}
		clock := clockwork.NewFakeClockAt(xt)
		fn := fileutil.GenerateFn(pattern, clock, 24*time.Hour)
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
