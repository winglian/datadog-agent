// +build !windows

package main

import (
	"github.com/DataDog/datadog-agent/cmd/process-agent/app"
)

func main() {
	app.Run()
}
