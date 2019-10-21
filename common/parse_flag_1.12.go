// +build !go1.13

package common

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"testing"
)

func ParseFlag() (err error) {

	restore := flag.CommandLine
	defer func() {
		flag.CommandLine = restore
	}()
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	// flag.CommandLine.Usage = func() {}
	var buf = new(bytes.Buffer)
	flag.CommandLine.SetOutput(buf)
	flag.CommandLine.StringVar(&ENVIRONMENT, "e", "", "set env, e.g dev test prod")
	if IsGoTest() {
		testing.Init()
	}

	err = flag.CommandLine.Parse(os.Args[1:])
	if err != nil {
		err = errors.New(buf.String())
	}

	if os.Getenv("VERSION") != "" {
		VERSION = os.Getenv("VERSION")
	}

	if len(ENVIRONMENT) == 0 {
		ENVIRONMENT = os.Getenv("ENVIRONMENT")
	}
	if len(ENVIRONMENT) == 0 && (IsGoRun() || IsGoTest()) {
		ENVIRONMENT = DEV
	}

	return
}
