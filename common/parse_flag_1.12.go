// +build !go1.13

package common

import (
	"flag"
	"os"
)

func ParseFlag() (err error) {

	flag.StringVar(&ENVIRONMENT, "e", "", "set env, e.g dev test prod")
	flag.Parse()

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
