// +build go1.13

package common

import (
	"testing"
)

func loadFlag() {
	if IsGoTest() {
		testing.Init()
	}
}
