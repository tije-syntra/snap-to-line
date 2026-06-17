package snaptoline

import "errors"

var (
	ErrInsufficientStops       = errors.New("snaptoline: at least two stops required")
	ErrInvalidLine             = errors.New("snaptoline: line must contain at least two points")
	ErrStopMeasureNotMonotonic = errors.New("snaptoline: projected stop measures must be monotonically increasing")
	ErrInvalidMeasureRange     = errors.New("snaptoline: invalid measure range")
	ErrNoCandidates            = errors.New("snaptoline: no snap candidates found")
	ErrEmptySegments           = errors.New("snaptoline: no segments built from stops")
)
