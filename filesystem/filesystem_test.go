package filesystem

import (
	"testing"

	"github.com/ecletus/oss/tests"
)

func TestAll(t *testing.T) {
	fileSystem := New("/tmp")
	tests.TestAll(fileSystem, t)
}
