package encoding

import (
	"testing"
)

func TestSnake(t *testing.T) {
	s := Snake("testTest")
	if s == "test_test" {
		t.Logf("Ok: want:test_test got:%s", s)
	} else {
		t.Fatalf("want:test_test got:%s", s)
	}
}
