package presentation

import (
	"encoding/binary"

	"github.com/olivierh59500/ikplus/internal/game"
)

// packedBitmap is a native drawing surface. Rows contain four 40-byte planes;
// retaining their bits preserves overlapping ink and ground-shadow operations.
type packedBitmap struct {
	pixels  []byte
	storage []byte
}

// Two-pixel alignment can spill a word before the visible surface. A small
// drawing margin retains those bits without addressing a preceding image.
func (b *packedBitmap) allocate(size int) {
	b.storage = make([]byte, size+4)
	b.pixels = b.storage[4:]
}

func (b *packedBitmap) pixel(offset int) byte           { return b.storage[offset+4] }
func (b *packedBitmap) putPixel(offset int, value byte) { b.storage[offset+4] = value }

func (b *packedBitmap) word(offset int) uint16 {
	return binary.BigEndian.Uint16(b.storage[offset+4:])
}
func (b *packedBitmap) long(offset int) uint32 {
	return binary.BigEndian.Uint32(b.storage[offset+4:])
}
func (b *packedBitmap) putWord(offset int, value uint16) {
	binary.BigEndian.PutUint16(b.storage[offset+4:], value)
}
func (b *packedBitmap) putLong(offset int, value uint32) {
	binary.BigEndian.PutUint32(b.storage[offset+4:], value)
}

// savedDrawing identifies a rectangle on a native surface for later erasure.
type savedDrawing struct {
	Picture *packedBitmap
	Offset  int
}

func (p *sceneryPlanes) screens() [2]*packedBitmap {
	return [2]*packedBitmap{&p.pictures[0], &p.pictures[1]}
}

func (p *sceneryPlanes) pictureBuffer(buffer game.PictureBuffer) *packedBitmap {
	if buffer == game.PictureFirst {
		return &p.pictures[0]
	}
	return &p.pictures[1]
}
