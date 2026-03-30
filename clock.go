package mediaforge

import "time"

// Clock abstracts time operations for testability.
type Clock interface {
	Now() time.Time
}

// realClock uses the standard time package.
type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }
