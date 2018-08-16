package filesystem

import (
	"testing"

	"github.com/aghape/oss/tests"
)

func TestAll(t *testing.T) {
	fileSystem := New("/tmp")
	tests.TestAll(fileSystem, t)
}
