/*
 * Copyright (C) 2023 by Jason Figge
 */

package internal

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/veandco/go-sdl2/sdl"
	"us.figge/guilib/graphics"
	"us.figge/guilib/graphics/fonts"
)

const (
	Red    = uint32(0xFF0000FF)
	Green1 = uint32(0x00FF00FF)
	Green2 = uint32(0x00CC00FF)
	Green3 = uint32(0x009900FF)
	Green4 = uint32(0x006600FF)
	Green5 = uint32(0x003300FF)
	Blue   = uint32(0x0000FFFF)

	StepSize = 2
)

type line struct {
	X1, Y1, X2, Y2 float32
	color          uint32
}

type rayCast struct {
	pos   *sdl.FPoint
	angle float64
}

type fov struct {
	width  int32
	heigth int32
}

type Controller struct {
	graphics.BaseHandler
	graphics.CoreMethods
	width      int32
	height     int32
	panelW     int32
	panelDelta float32
	panelDiag  float32
	fovRayCnt  int32
	fovDegrees int32
	fovDelta   float64
	fovStart   float64
	fovDStart  float64
	walls      []*line
	ray        *rayCast
}

func NewController(width int32, height int32) *Controller {
	W := float32(width / 2)
	H := float32(height)
	c := &Controller{
		width:     width,
		height:    height,
		panelDiag: float32(math.Sqrt(float64(W*W + H*H))),
		walls: []*line{
			{X1: 0, Y1: 1, X2: W, Y2: 1, color: Blue},
			{X1: W, Y1: 0, X2: W, Y2: H - 1, color: Blue},
			{X1: W, Y1: H - 1, X2: 0, Y2: H - 1, color: Blue},
			{X1: 0, Y1: H - 1, X2: 0, Y2: 1, color: Blue},
			{X1: 161, Y1: 307, X2: 274, Y2: 342, color: Green1},
			{X1: 167, Y1: 423, X2: 180, Y2: 331, color: Green2},
			{X1: 439, Y1: 382, X2: 437, Y2: 537, color: Green3},
			{X1: 86, Y1: 246, X2: 348, Y2: 45, color: Green4},
			{X1: 432, Y1: 86, X2: 281, Y2: 69, color: Green5},
		},
		ray: &rayCast{
			pos:   &sdl.FPoint{X: W / 2, Y: H / 2},
			angle: 0,
		},
		fovRayCnt:  100,
		fovDegrees: 60,
		panelW:     width / 2,
	}
	c.fovDelta = float64(c.fovDegrees) * math.Pi / 180 / float64(c.fovRayCnt)
	c.fovStart = c.ray.angle - c.fovDelta*float64(c.fovRayCnt/2)
	c.fovDStart = c.fovStart
	c.panelDelta = float32(c.panelW / c.fovRayCnt)

	return c
}

func (c *Controller) Init(canvas *graphics.Canvas) {
	fonts.LoadFonts(canvas.Renderer())
	graphics.ErrorTrap(canvas.Renderer().SetDrawBlendMode(sdl.BLENDMODE_BLEND))
	c.AddDestroyer(fonts.FreeFonts)
}

func (c *Controller) Events(event sdl.Event) bool {
	processed := false
	switch e := event.(type) {
	case *sdl.MouseButtonEvent:
		processed = c.mouseButtonEvent(e)
	case *sdl.MouseMotionEvent:
		processed = c.mouseMotionEvent(e)
	case *sdl.KeyboardEvent:
		processed = c.keyboardEvent(e)
	}
	return processed
}

func (c *Controller) OnUpdate() {
	codes := sdl.GetKeyboardState()
	if codes[sdl.SCANCODE_W] == 1 {
		c.forward()
	} else if codes[sdl.SCANCODE_S] == 1 {
		c.backwards()
	}

	if codes[sdl.SCANCODE_A] == 1 {
		c.strafe(true)
	} else if codes[sdl.SCANCODE_D] == 1 {
		c.strafe(false)
	}
}

func (c *Controller) OnDraw(renderer *sdl.Renderer) {
	graphics.ErrorTrap(c.Clear(renderer, uint32(0)))
	c.drawRays(renderer)
	c.drawWalls(renderer)
	graphics.ErrorTrap(c.WriteFrameRate(renderer, c.width-115, 0))
}

func (c *Controller) drawWalls(renderer *sdl.Renderer) {
	for _, wall := range c.walls {
		graphics.ErrorTrap(renderer.SetDrawColor(uint8(wall.color>>24), uint8(wall.color>>16), uint8(wall.color>>8), uint8(wall.color)))
		_ = renderer.DrawLineF(wall.X1, wall.Y1, wall.X2, wall.Y2)
	}
}

func (c *Controller) drawRays(renderer *sdl.Renderer) {
	var closestPt *sdl.FPoint
	var closestWall *line
	r := c.fovStart
	fr := c.fovDStart
	rect := &sdl.FRect{X: float32(c.panelW), Y: 0, W: c.panelDelta, H: 0}
	for i := int32(0); i < c.fovRayCnt; i++ {
		// Calculate the distance to a wall for each ray
		pt := c.translatePoint(c.ray.pos, r, 8)
		closestPt = nil
		distance := math.MaxFloat64
		for _, wall := range c.walls {
			pt2 := c.rayToWallIntersect(c.ray.pos.X, c.ray.pos.Y, pt.X, pt.Y, wall.X1, wall.Y1, wall.X2, wall.Y2)
			if pt2 != nil { // intersects with wall
				rayLength := c.lineLength(c.ray.pos.X, c.ray.pos.Y, pt2.X, pt2.Y)
				if rayLength < distance {
					distance = rayLength
					closestPt = pt2
					closestWall = wall
				}
			}
		}
		graphics.ErrorTrap(renderer.SetDrawColor(0xFF, 0xFF, 0xFF, 0x16))
		graphics.ErrorTrap(renderer.DrawLineF(c.ray.pos.X, c.ray.pos.Y, closestPt.X, closestPt.Y))

		//rect.H = graphics.FMap(float32(distance*math.Cos(fr)), 0, c.panelDiag, float32(c.height), 0)
		rect.H = graphics.FMap(float32(distance*math.Cos(fr)), 20, c.panelDiag, 800, 100)
		if rect.H > 0 {
			rect.Y = float32(c.height)/2 - rect.H/2
			//colorAlpha := uint8(graphics.FMap(rect.H*rect.H, float32(c.height*c.height), 0, 255, 0))
			colorAlpha := uint8(255)
			graphics.ErrorTrap(renderer.SetDrawColor(uint8(closestWall.color>>24), uint8(closestWall.color>>16), uint8(closestWall.color>>8), colorAlpha))
			graphics.ErrorTrap(renderer.FillRectF(rect))
		}

		r += c.fovDelta
		fr += c.fovDelta
		rect.X += c.panelDelta
	}

}

func (c *Controller) translatePoint(origin *sdl.FPoint, t float64, length float64) *sdl.FPoint {
	return &sdl.FPoint{
		X: origin.X + float32(length*math.Sin(t)),
		Y: origin.Y - float32(length*math.Cos(t)),
	}
}

func (c *Controller) lineIntersect(X1, Y1, X2, Y2, X3, Y3, X4, Y4 float32) (float32, float32, bool) {
	den := (X1-X2)*(Y3-Y4) - (Y1-Y2)*(X3-X4)
	if den == 0 {
		return 0, 0, false
	}
	t := ((X1-X3)*(Y3-Y4) - (Y1-Y3)*(X3-X4)) / den
	u := ((X1-X3)*(Y1-Y2) - (Y1-Y3)*(X1-X2)) / den
	return t, u, true
}

func (c *Controller) rayToWallIntersect(X1, Y1, X2, Y2, X3, Y3, X4, Y4 float32) *sdl.FPoint {
	if t, u, ok := c.lineIntersect(X1, Y1, X2, Y2, X3, Y3, X4, Y4); ok && 0 <= u && u <= 1 && t > 0 {
		return &sdl.FPoint{
			X: X1 + t*(X2-X1),
			Y: Y1 + t*(Y2-Y1),
		}
	}
	return nil
}

func (c *Controller) wallIntersect(X1, Y1, X2, Y2, X3, Y3, X4, Y4 float32) bool {
	t, u, ok := c.lineIntersect(X1, Y1, X2, Y2, X3, Y3, X4, Y4)
	return ok && 0 <= u && u <= 1 && 0 <= t && t <= 1
}

func (c *Controller) moveRay(pt *sdl.FPoint) bool {
	for _, wall := range c.walls {
		if c.wallIntersect(c.ray.pos.X, c.ray.pos.Y, pt.X, pt.Y, wall.X1, wall.Y1, wall.X2, wall.Y2) {
			return false
		}
	}
	c.ray.pos = pt
	return true
}

func (c *Controller) lineLength(X1, Y1, X2, Y2 float32) float64 {
	return math.Sqrt(float64((X2-X1)*(X2-X1) + (Y2-Y1)*(Y2-Y1)))
}

func (c *Controller) mouseDirection(X1, Y1, X2, Y2 float32) float64 {
	return math.Mod(-math.Atan2(float64(X2-X1), float64(Y2-Y1)), 2*math.Pi) + math.Pi
}

func (c *Controller) setFovAngle(X int32, Y int32) {
	c.ray.angle = c.mouseDirection(c.ray.pos.X, c.ray.pos.Y, float32(X), float32(Y))
	c.fovStart = c.ray.angle - c.fovDelta*float64(c.fovRayCnt/2)
}

func (c *Controller) strafe(clockwise bool) {
	X, Y, state := sdl.GetMouseState()
	if state == 0 {
		offset := math.Pi / 2
		if clockwise {
			offset = -offset
		}
		c.moveRay(c.translatePoint(c.ray.pos, c.ray.angle+offset, 2))
	} else {
		r := c.lineLength(c.ray.pos.X, c.ray.pos.Y, float32(X), float32(Y))
		if r == 0 {
			return
		}
		t := float64(StepSize) / r
		angle := c.ray.angle + t
		if !clockwise {
			angle = c.ray.angle - t
		}
		o := math.Sin(angle) * r
		a := math.Cos(angle) * r
		pt := &sdl.FPoint{X: float32(X) - float32(o), Y: float32(Y) + float32(a)}
		if c.moveRay(pt) {
			c.setFovAngle(X, Y)
			c.ray.angle = angle
		}
	}
}

func (c *Controller) forward() {
	X, Y, state := sdl.GetMouseState()
	if state == 1 && math.Abs(float64(X-int32(c.ray.pos.X))+math.Abs(float64(Y-int32(c.ray.pos.Y)))) <= 2 {
		c.moveRay(&sdl.FPoint{X: float32(X), Y: float32(Y)})
	} else {
		c.moveRay(c.translatePoint(c.ray.pos, c.ray.angle, 2))
	}
}

func (c *Controller) backwards() {
	c.moveRay(c.translatePoint(c.ray.pos, c.ray.angle+math.Pi, StepSize))
}

func (c *Controller) mouseButtonEvent(event *sdl.MouseButtonEvent) bool {
	if event.State == 1 {
		if event.Button == 3 {
			c.ray.pos.X = float32(event.X)
			c.ray.pos.Y = float32(event.Y)
		} else {
			c.setFovAngle(event.X, event.Y)
		}
		return true
	}
	return false
}

func (c *Controller) mouseMotionEvent(event *sdl.MouseMotionEvent) bool {
	if event.State != 0 {
		c.setFovAngle(event.X, event.Y)
		return true
	}
	return false
}

func (c *Controller) keyboardEvent(event *sdl.KeyboardEvent) bool {
	if event.State == 1 {
		if event.Keysym.Scancode == sdl.SCANCODE_Q {
			c.Quit()
		} else if event.Keysym.Scancode == sdl.SCANCODE_P {
			c.walls = c.walls[:4]
			for i := 0; i < 5; i++ {
				c.walls = append(c.walls, &line{
					X1: float32(rand.Int31n(c.width / 2)),
					Y1: float32(rand.Int31n(c.height)),
					X2: float32(rand.Int31n(c.width / 2)),
					Y2: float32(rand.Int31n(c.height)),
				})
				fmt.Printf(
					"{X1:%d, Y1:%d, X2:%d, Y2:%d},\n",
					int(c.walls[4+i].X1), int(c.walls[4+i].Y1), int(c.walls[4+i].X2), int(c.walls[4+i].Y2))
			}
			fmt.Printf("---------------\n\n")
		}
		return true
	}
	return false
}
