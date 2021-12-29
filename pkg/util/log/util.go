package log

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/DataDog/datadog-agent/pkg/util/scrubber"
)

func buildLogEntry(v ...interface{}) string {
	var fmtBuffer bytes.Buffer

	for i := 0; i < len(v)-1; i++ {
		fmtBuffer.WriteString("%v ")
	}
	fmtBuffer.WriteString("%v")

	return fmt.Sprintf(fmtBuffer.String(), v...)
}

func scrubbedError(message string) error {
	msgScrubbed, err := scrubber.ScrubBytes([]byte(message))
	if err == nil {
		return errors.New(string(msgScrubbed))
	}
	return errors.New("[REDACTED] - failure to clean the message")
}
