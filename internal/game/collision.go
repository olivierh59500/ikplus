package game

// The authored contact tables are extracted offline. The original contact
// rule checks six body categories against a strike mask and
// compares their horizontal points in the Amiga's two-pixel position units.
type collisionSource struct {
	frames, tips, masks, widths []byte
	poses, bodies, rear         []byte
	reactions                   []byte
}

var sourceCollisions = loadCollisions()

func loadCollisions() collisionSource {
	data := &nativeGameplay.Collision
	return collisionSource{data.Frames, data.Tips, data.Masks, data.Widths, data.Poses, data.Bodies, data.Rear, data.Reactions}
}

// bodyCenter converts the original asymmetric drawing anchor to the standing
// torso center. Subtracting the display's32-pixel drawing bias leaves the same
// relative distances as the original $7402 orientation routine.
func bodyCenter(f Fighter) float64 {
	if f.Facing < 0 {
		return f.X + 24
	}
	return f.X - 8
}

func poseFrame(f Fighter) int {
	if f.computerAnimationReady && int(f.computerAnimation.Action) == f.Action {
		return int(f.computerAnimation.Frame)
	}
	if f.SpriteFrame != 0 || f.Move == Idle {
		return f.SpriteFrame
	}
	action := sourceAction(f.Move)
	if f.lastInput.hasOriginalAction {
		action = f.Action
	}
	sequence := sourceAnimations[action]
	step := min(max(0, (f.Age-1)/AnimationTicks), len(sequence.Frames)-1)
	return sequence.Frames[step]
}

func strikeIndex(frame int) int {
	for index, value := range sourceCollisions.frames {
		if int(value) == frame {
			return index
		}
	}
	return -1
}

// Strike outcomes follow the authored visible frame and impact orientation.
// Scores are stored as BCD hundreds; reaction timers are ordinary update counts.
func strikeAward(index int, front bool) (score, stun int) {
	c := &nativeGameplay.Collision
	value, timer := c.RearScores[index], c.RearStuns[index]
	if front {
		value, timer = c.FrontScores[index], c.FrontStuns[index]
	}
	return (int(value>>4)*10 + int(value&15)) * 100, int(timer)
}

// Original $7666 recognizes attacks by their visible strike frame, including
// byte action aliases whose nominal action is outside the extracted catalog.
func sourceStrikeMove(frame int) (Move, bool) {
	switch frame {
	case 14:
		return StomachKick, true
	case 16:
		return FaceKick, true
	case 18:
		return CrouchPunch, true
	case 20:
		return FrontPunch, true
	case 21:
		return ShinKick, true
	case 27:
		return Headbutt, true
	case 32:
		return Sweep, true
	case 67, 68:
		return DoubleKick, true
	case 69:
		return FlyingKick, true
	default:
		return Idle, false
	}
}

func sourceContact(a, t Fighter, frame int) (int, bool) {
	index := strikeIndex(frame)
	if index < 0 {
		return 0, false
	}
	targetFrame := poseFrame(t)
	if targetFrame < 0 || targetFrame >= len(sourceCollisions.poses) {
		return 0, false
	}
	pose := int(sourceCollisions.poses[targetFrame])
	if pose >= 128 {
		return 0, false
	}
	if pose == 23 && index == 2 {
		pose = 35
	}
	aPosition, tPosition := byte(int(a.X/2)), byte(int(t.X/2))
	aCenter, tCenter := aPosition+12, tPosition+12
	if a.Facing < 0 {
		aCenter += 16
	}
	if t.Facing < 0 {
		tCenter += 16
	}
	orientation := 0
	if int8(aCenter-tCenter) >= 0 {
		orientation |= 4
	}
	if a.Facing < 0 {
		orientation |= 2
	}
	if t.Facing < 0 {
		orientation |= 1
	}
	if pose == 19 && (orientation == 0 || orientation == 7) {
		pose = 0
	}
	tip := sourceCollisions.tips[index]
	if a.Facing < 0 {
		tip = 40 - tip
	}
	attackPoint := aPosition + tip
	width := sourceCollisions.widths[index]
	for category := 0; category < 6; category++ {
		if sourceCollisions.masks[index]&(32>>category) == 0 {
			continue
		}
		body := int(sourceCollisions.bodies[pose*6+category])
		if body >= 128 {
			continue
		}
		if t.Facing < 0 {
			body = 40 - body
		}
		bodyPoint := tPosition + byte(body)
		delta := byte(0)
		switch orientation & 3 {
		case 0:
			delta = attackPoint - bodyPoint + width
		case 1:
			delta = attackPoint - bodyPoint
		case 2:
			delta = bodyPoint - attackPoint
		case 3:
			delta = bodyPoint - attackPoint + width
		}
		if int8(delta) >= 0 && delta <= width {
			return orientation, true
		}
	}
	return orientation, false
}
