package game

import (
	"bytes"
	"fmt"
	"image"
	"image/png"

	"github.com/olivierh59500/ikplus/assets"
	"github.com/olivierh59500/ikplus/internal/graphics"
)

type bonusPixelMask struct {
	width, height, words int
	rows                 []uint16
}

func (m bonusPixelMask) at(x, y int) bool {
	return x >= 0 && y >= 0 && x < m.width && y < m.height && m.rows[y*m.words+x/16]&(0x8000>>uint(x&15)) != 0
}

func bonusImage(name string) image.Image {
	raw, err := assets.Read("original/" + name + ".png")
	if err != nil {
		panic(fmt.Sprintf("read original bonus mask %s: %v", name, err))
	}
	picture, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		panic(fmt.Sprintf("decode original bonus mask %s: %v", name, err))
	}
	return picture
}

func imagePixelMask(picture image.Image, region image.Rectangle) bonusPixelMask {
	m := bonusPixelMask{width: region.Dx(), height: region.Dy(), words: (region.Dx() + 15) / 16}
	m.rows = make([]uint16, m.words*m.height)
	for y := 0; y < m.height; y++ {
		for x := 0; x < m.width; x++ {
			_, _, _, alpha := picture.At(region.Min.X+x, region.Min.Y+y).RGBA()
			if alpha != 0 {
				m.rows[y*m.words+x/16] |= 0x8000 >> uint(x&15)
			}
		}
	}
	return m
}

var bonusPixelMasks = func() struct {
	fighters     [88]bonusPixelMask
	shields      map[string]bonusPixelMask
	balls        [8][256]bonusPixelMask
	fragments    [4][8]bonusPixelMask
	bomb, marker bonusPixelMask
} {
	var masks struct {
		fighters     [88]bonusPixelMask
		shields      map[string]bonusPixelMask
		balls        [8][256]bonusPixelMask
		fragments    [4][8]bonusPixelMask
		bomb, marker bonusPixelMask
	}
	atlas := bonusImage("sprites-white")
	for frame := range masks.fighters {
		x, y := (frame%12)*96, (frame/12)*80
		masks.fighters[frame] = imagePixelMask(atlas, image.Rect(x, y, x+96, y+80))
	}
	masks.shields = make(map[string]bonusPixelMask)
	for _, name := range []string{"shield-left", "shield-right", "shield-neutral", "shield-neutral-alternate"} {
		picture := bonusImage(name)
		masks.shields[name] = imagePixelMask(picture, picture.Bounds())
	}
	artwork := graphics.MustLoad()
	for color := range masks.balls {
		for frame := range masks.balls[color] {
			cell := artwork.Sprites[artwork.Balls[color][frame]]
			mask := bonusPixelMask{width: 16, height: len(cell.Rows), words: 1, rows: make([]uint16, len(cell.Rows))}
			for row, planes := range cell.Rows {
				mask.rows[row] = planes[0] | planes[1] | planes[2] | planes[3]
			}
			masks.balls[color][frame] = mask
		}
	}
	picture := bonusImage("bomb-body")
	masks.bomb = imagePixelMask(picture, picture.Bounds())
	picture = bonusImage("kick-marker")
	masks.marker = imagePixelMask(picture, picture.Bounds())
	picture = bonusImage("bomb-fragments")
	for variant := range masks.fragments {
		for frame := range masks.fragments[variant] {
			masks.fragments[variant][frame] = imagePixelMask(picture, image.Rect(frame*16, variant*9, frame*16+16, variant*9+9))
		}
	}
	return masks
}()

func projectilePixelMask(color, spin int) bonusPixelMask {
	return bonusPixelMasks.balls[color&7][byte(spin)]
}

// ShieldAssetName follows the original $B9AE pointer selection. Rendering and
// collision use the same authored cells, including the two-board neutral pose.
func ShieldAssetName(shieldType int, alternate bool) string {
	switch shieldType {
	case 0:
		return "shield-left"
	case 2:
		return "shield-right"
	default:
		if alternate {
			return "shield-neutral-alternate"
		}
		return "shield-neutral"
	}
}

// KickMarker returns the source $C88A marker origin. Only frames 21/32 with
// held counter below three publish the one-row weapon mask at scanline 184.
func KickMarker(f Fighter) (x, y int, active bool) {
	if (f.SpriteFrame != 21 && f.SpriteFrame != 32) || f.computerAnimation.Held >= 3 {
		return 0, 0, false
	}
	width := int(nativeGameplay.Animation.Widths[f.SpriteFrame])
	x = int(f.X) + width*2 - 40
	if f.Facing < 0 {
		x = int(f.X) + 50 - width*2
	}
	return x, 184, true
}

func bonusBodyPixel(f Fighter, x, y int) bool {
	if x < 0 || x >= Width || y < 0 || y >= Height || f.SpriteFrame < 0 || f.SpriteFrame >= 88 {
		return false
	}
	left := int(f.X) - 32
	if f.Facing < 0 {
		left = int(f.X) - 48
	}
	localX, localY := x-left, y-116
	// The source RLE painter writes packed mask words without an additional
	// horizontal clip. A few authored poses extend past their clamp width;
	// preserve their word spill into the adjacent mask row.
	if localX < 0 {
		localX += Width
		localY--
	} else if localX >= Width {
		localX -= Width
		localY++
	}
	if f.Facing < 0 {
		localX = 95 - localX
	}
	return bonusPixelMasks.fighters[f.SpriteFrame].at(localX, localY)
}

func bonusWeaponPixel(f Fighter, shield, alternate bool, x, y int) bool {
	if x < 0 || x >= Width || y < 0 || y >= Height {
		return false
	}
	if shield {
		return bonusPixelMasks.shields[ShieldAssetName(f.ShieldType, alternate)].at(x-int(f.ShieldX), y-int(f.ShieldY))
	}
	left, top, active := KickMarker(f)
	return active && bonusPixelMasks.marker.at(x-left, y-top)
}

// bonusMaskContact reproduces the original two auxiliary collision planes.
// Object positions are integer display coordinates after source 8.8 truncation.
func bonusMaskContact(f Fighter, shield bool, object bonusPixelMask, left, top int) (body, weapon bool) {
	// $1BDBA/$1C076 convert a packed four-plane framebuffer address back
	// to a one-plane mask address. Keep that conversion, including the
	// source's three-row offset for objects crossing the left screen edge.
	x, y := (left+32)&511, top&255
	if y >= Height {
		return
	}
	aligned, shift := (x>>1)&248, uint(x&15)
	if aligned < 8 || aligned >= 176 {
		return
	}
	clip := uint32(0xffffffff)
	if aligned < 16 {
		clip = 0x0000ffff
	} else if aligned >= 168 {
		clip = 0xffff0000
	}
	packed := (y*160 + aligned/4 - 4) & 32766
	maskOffset := (packed/160)*40 + packed%160
	for row := 0; row < min(object.height, Height-y); row++ {
		bits := uint32(object.rows[row*object.words]) << 16
		if shift != 0 {
			bits = bits>>shift | bits<<(32-shift)
		}
		bits &= clip
		for bit := 0; bit < 32; bit++ {
			if bits&(0x80000000>>uint(bit)) == 0 {
				continue
			}
			address := (maskOffset+row*40)*8 + bit
			px, py := address%Width, address/Width
			body = body || bonusBodyPixel(f, px, py)
			weapon = weapon || bonusWeaponPixel(f, shield, f.ShieldAlternate, px, py)
			if body && weapon {
				return
			}
		}
	}
	return
}

// ShieldPoseContact exposes static authored geometry, not future match state.
// The tool-assisted player combines it with its own observed trajectory model.
func ShieldPoseContact(state, color, spin, x, y int) (body, weapon bool) {
	if state < 0 || state >= 7 || x+24 < 119 || x+24 >= 270 {
		return
	}
	s := shieldControls.States[state]
	f := Fighter{X: float64(s.FighterX&248) * 2, Y: GroundY, Facing: 1, SpriteFrame: s.Frame, ShieldType: s.Type, ShieldX: float64(s.ShieldX&248)*2 - 32, ShieldY: float64(s.Y)}
	if s.Facing != 0 {
		f.Facing = -1
	}
	return bonusMaskContact(f, true, projectilePixelMask(color, spin), x-8, y-6)
}

// FragmentPoseContact queries static sprite artwork at a supplied pose. It has
// no access to live actors, projectile motion, animation commands, or RNG.
func FragmentPoseContact(frame, facing, anchorX, variant, spin, x, y int) bool {
	f := Fighter{X: float64(anchorX), Facing: facing, SpriteFrame: frame}
	body, _ := bonusMaskContact(f, false, bonusPixelMasks.fragments[variant&3][spin&7], x-8, y-6)
	return body
}
