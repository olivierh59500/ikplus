package game

// PictureBuffer identifies a retained drawing surface owned by the renderer.
type PictureBuffer uint8

const (
	PictureNone PictureBuffer = iota
	PictureFirst
	PictureSecond
)

// CoveragePlane identifies one normal or mirrored fighter coverage surface.
type CoveragePlane uint8

const (
	CoverageNone CoveragePlane = iota
	CoverageFirst
	CoverageSecond
	CoverageFirstMirrored
	CoverageSecondMirrored
)

// RestorationCursor describes saved rectangles in a drawing surface.
type RestorationCursor struct {
	Buffer PictureBuffer
	Count  uint16
}
