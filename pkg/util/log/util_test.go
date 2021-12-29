package log

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScrubbedError(t *testing.T) {
	setupScrubbing(t)

	err := scrubbedError("a SECRET message")
	assert.Equal(t, "a ****** message", err.Error())
}
