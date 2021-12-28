package log

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatErrorfScrubbing(t *testing.T) {
	setupScrubbing(t)

	err := formatErrorf("%s", "a SECRET message")
	assert.Equal(t, "a ****** message", err.Error())
}

func TestFormatErrorScrubbing(t *testing.T) {
	setupScrubbing(t)

	err := formatError("a big SECRET")
	assert.Equal(t, "a big ******", err.Error())
}
