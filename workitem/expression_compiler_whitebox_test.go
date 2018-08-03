package workitem

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnsureJoinTable(t *testing.T) {
	ec := newExpressionCompiler()

	assert.False(t, ec.joins["iteration"].Active)
	ec.ensureJoinTable("iterations")
	assert.True(t, ec.joins["iteration"].Active)

	assert.False(t, ec.joins["area"].Active)
	ec.ensureJoinTable("areas")
	assert.True(t, ec.joins["area"].Active)
}
