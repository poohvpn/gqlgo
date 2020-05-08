package gqlgo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPath(t *testing.T) {
	as := assert.New(t)
	as.Equal("variables.var1", getPath(true, 0, "var1"))
	as.Equal("variables.var1.0", getPath(true, 0, "var1", 0))
	as.Equal("variables.var1.0", getPath(true, 0, "var1", 0, 0))
	as.Equal("0.variables.var1", getPath(false, 0, "var1"))
	as.Equal("1.variables.var1.1", getPath(false, 1, "var1", 1))
}
