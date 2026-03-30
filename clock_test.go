package mediaforge

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRealClock_Now(t *testing.T) {
	c := realClock{}
	before := time.Now()
	got := c.Now()
	after := time.Now()

	assert.False(t, got.Before(before), "clock.Now() should not be before time.Now()")
	assert.False(t, got.After(after), "clock.Now() should not be after time.Now()")
}

func TestRealClock_ImplementsClock(t *testing.T) {
	var _ Clock = realClock{}
}
