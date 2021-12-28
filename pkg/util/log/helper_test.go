package log

import (
	"regexp"
	"testing"

	"github.com/DataDog/datadog-agent/pkg/util/scrubber"
)

// Set up for scrubbing tests, by temporarily setting Scrubber; this avoids testing
// the default scrubber's functionality in this module
func setupScrubbing(t *testing.T) {
	oldScrubber := scrubber.DefaultScrubber
	scrubber.DefaultScrubber = scrubber.New()
	scrubber.DefaultScrubber.AddReplacer(scrubber.SingleLine, scrubber.Replacer{
		Regex: regexp.MustCompile("SECRET"),
		Repl:  []byte("******"),
	})
	t.Cleanup(func() { scrubber.DefaultScrubber = oldScrubber })
}
