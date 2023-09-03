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
	Green1 = uint32(0x00FF00FF)
	Green2 = uint32(0x99CC99FF)
	Green3 = uint32(0x99ccFF)
	Green4 = uint32(0xcc6699FF)
	Green5 = uint32(0x993366FF)
	Blue   = uint32(0x0000FFFF)

	StepSize = 2
)

const (
	ArenaWidth    = 600
	ArenaHeight   = 600
	WallHeight    = 200
	PLayerStartX  = 300
	PlayerStartY  = 300
	PlayerHeight  = 100
	PortalWidth   = 320
	HorizontalFOV = 60
	VerticalFOV   = 45
	RayCount      = 50
)

var WallColors = []uint32{Green1, Green2, Green3, Green4, Green5}

type ClosestEntity struct {
	X     float32
	Y     float32
	color uint32
}

type Wall struct {
	X1, Y1, X2, Y2 float32
	color          uint32
}

type Player struct {
	position  *sdl.FPoint
	height    int32
	direction float64
	fov       *Fov
}

type Fov struct {
	portalWidth      int32
	portalHeight     float32
	portalRect       *sdl.FRect
	portalDistance   float64
	portalBrightness float32
	portalDelta      float32
	fisheyeDelta     float64
	hFOV             float64
	vFOV             float64
	RayDelta         float64
	RayStart         float64
	RayEnd           float64
}

type Controller struct {
	graphics.BaseHandler
	graphics.CoreMethods
	arenaWidth  int32
	arenaHeight int32
	WallHeight  float64
	walls       []*Wall
	player      Player
	shiftDown   bool
}

func NewController() *Controller {
	c := &Controller{
		arenaWidth:  ArenaWidth,
		arenaHeight: ArenaHeight,
		WallHeight:  WallHeight,
		player: Player{
			position: &sdl.FPoint{
				X: PLayerStartX,
				Y: PlayerStartY,
			},
			height:    PlayerHeight,
			direction: 0,
			fov: &Fov{
				portalWidth: PortalWidth,
				hFOV:        (HorizontalFOV * math.Pi) / 180,
				vFOV:        (VerticalFOV * math.Pi) / 180,
			},
		},
		walls: []*Wall{
			{X1: 0, Y1: 1, X2: ArenaWidth, Y2: 1, color: Blue},
			{X1: ArenaWidth, Y1: 0, X2: ArenaWidth, Y2: ArenaHeight - 1, color: Blue},
			{X1: ArenaWidth, Y1: ArenaHeight - 1, X2: 0, Y2: ArenaHeight - 1, color: Blue},
			{X1: 0, Y1: ArenaHeight - 1, X2: 0, Y2: 1, color: Blue},
			{X1: 161, Y1: 307, X2: 274, Y2: 342, color: Green1},
			{X1: 167, Y1: 423, X2: 180, Y2: 331, color: Green2},
			{X1: 439, Y1: 382, X2: 437, Y2: 537, color: Green3},
			{X1: 86, Y1: 246, X2: 348, Y2: 45, color: Green4},
			{X1: 432, Y1: 86, X2: 281, Y2: 69, color: Green5},
		},
	}
	c.player.fov.portalDistance = float64(c.player.fov.portalWidth/2) / math.Tan(c.player.fov.hFOV/2)
	c.player.fov.portalBrightness = float32(c.player.fov.portalDistance * c.player.fov.portalDistance * 2.25)
	c.player.fov.portalHeight = float32(math.Floor(math.Tan(c.player.fov.vFOV/2) * c.player.fov.portalDistance * 2))
	c.player.fov.portalRect = &sdl.FRect{
		X: float32(ArenaWidth + ArenaWidth/2 - c.player.fov.portalWidth/2 - 1),
		Y: ArenaHeight/2 - c.player.fov.portalHeight/2 - 1,
		W: float32(c.player.fov.portalWidth + 2),
		H: c.player.fov.portalHeight + 2,
	}
	c.player.fov.portalDelta = float32(c.player.fov.portalWidth) / float32(RayCount)
	c.player.fov.fisheyeDelta = -(HorizontalFOV * math.Pi) / 360
	c.player.fov.RayDelta = c.player.fov.hFOV / float64(RayCount)
	c.player.fov.RayStart = c.player.direction - (HorizontalFOV*math.Pi)/360 + float64(c.player.fov.RayDelta)/2
	c.player.fov.RayEnd = c.player.direction + (HorizontalFOV*math.Pi)/360

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

	if c.shiftDown {
		if codes[sdl.SCANCODE_A] == 1 {
			c.strafe(true)
		} else if codes[sdl.SCANCODE_D] == 1 {
			c.strafe(false)
		}
	} else {
		if codes[sdl.SCANCODE_A] == 1 {
			c.rotate(false)
		} else if codes[sdl.SCANCODE_D] == 1 {
			c.rotate(true)
		}
	}
}

func (c *Controller) OnDraw(renderer *sdl.Renderer) {
	graphics.ErrorTrap(c.Clear(renderer, uint32(0x232323)))
	c.drawRays(renderer)
	c.drawWalls(renderer)
	c.drawPortal(renderer)
	graphics.ErrorTrap(c.WriteFrameRate(renderer, c.arenaWidth-115, 0))
}

func (c *Controller) drawWalls(renderer *sdl.Renderer) {
	for _, wall := range c.walls {
		graphics.ErrorTrap(renderer.SetDrawColor(uint8(wall.color>>24), uint8(wall.color>>16), uint8(wall.color>>8), uint8(wall.color)))
		graphics.ErrorTrap(renderer.DrawLineF(wall.X1, wall.Y1, wall.X2, wall.Y2))
	}
}

func (c *Controller) drawPortal(renderer *sdl.Renderer) {
	graphics.ErrorTrap(renderer.SetDrawColor(uint8(0xff), uint8(0), uint8(0), uint8(0xff)))
	graphics.ErrorTrap(renderer.DrawRectF(c.player.fov.portalRect))

	X1 := c.player.position.X + float32(math.Sin(c.player.fov.RayStart-c.player.fov.RayDelta/2)*c.player.fov.portalDistance)
	Y1 := c.player.position.Y - float32(math.Cos(c.player.fov.RayStart-+c.player.fov.RayDelta/2)*c.player.fov.portalDistance)
	X2 := c.player.position.X + float32(math.Sin(c.player.fov.RayEnd)*c.player.fov.portalDistance)
	Y2 := c.player.position.Y - float32(math.Cos(c.player.fov.RayEnd)*c.player.fov.portalDistance)
	graphics.ErrorTrap(renderer.SetDrawColor(uint8(0xff), uint8(0), uint8(0), uint8(0x80)))
	graphics.ErrorTrap(renderer.DrawLineF(X1, Y1, X2, Y2))
}

func (c *Controller) drawRays(renderer *sdl.Renderer) {
	closestEntity := ClosestEntity{}
	portalRect := &sdl.FRect{X: c.player.fov.portalRect.X + 1, Y: c.player.fov.portalRect.Y + 1, W: c.player.fov.portalDelta, H: c.player.fov.portalHeight}
	r := c.player.fov.RayStart
	fr := c.player.fov.fisheyeDelta
	for i := int32(0); i < RayCount; i++ {
		// Calculate the distance to a wall for each ray
		pt := c.translatePoint(c.player.position, r, 8)
		distance := math.MaxFloat64
		for _, wall := range c.walls {
			pt2 := c.rayToWallIntersect(c.player.position.X, c.player.position.Y, pt.X, pt.Y, wall.X1, wall.Y1, wall.X2, wall.Y2)
			if pt2 != nil { // intersects with wall
				rayLength := c.lineLength(c.player.position.X, c.player.position.Y, pt2.X, pt2.Y)
				if rayLength < distance {
					distance = rayLength
					closestEntity.X = pt2.X
					closestEntity.Y = pt2.Y
					closestEntity.color = wall.color
				}
			}
		}

		// Draw the rays until they strike the a wall
		graphics.ErrorTrap(renderer.SetDrawColor(0xFF, 0xFF, 0xFF, 0x16))
		graphics.ErrorTrap(renderer.DrawLineF(c.player.position.X, c.player.position.Y, closestEntity.X, closestEntity.Y))

		// Calculate the 3D column
		distance = math.Cos(fr) * distance
		portalRect.H = c.max(float32(c.player.fov.portalDistance/distance*c.WallHeight), c.player.fov.portalHeight)
		portalRect.Y = c.player.fov.portalRect.Y + 1 + c.player.fov.portalHeight/2 - portalRect.H/2

		// Determine color
		fColor := graphics.FMap(float32(distance*distance), 0, c.player.fov.portalBrightness, 255, 0)
		if fColor > 0 {
			// Render 3D column
			graphics.ErrorTrap(renderer.SetDrawColor(uint8(closestEntity.color>>24), uint8(closestEntity.color>>16), uint8(closestEntity.color>>8), uint8(fColor)))
			graphics.ErrorTrap(renderer.FillRectF(portalRect))
		}

		// Advance to next ray location
		r += c.player.fov.RayDelta
		fr += c.player.fov.RayDelta
		portalRect.X += c.player.fov.portalDelta
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

func (c *Controller) lineLength(X1, Y1, X2, Y2 float32) float64 {
	return math.Sqrt(float64((X2-X1)*(X2-X1) + (Y2-Y1)*(Y2-Y1)))
}

func (c *Controller) setFovAngle(X int32, Y int32) {
	c.player.direction = c.mouseDirection(c.player.position.X, c.player.position.Y, float32(X), float32(Y))
	c.player.fov.RayStart = c.player.direction - (HorizontalFOV*math.Pi)/360 + c.player.fov.RayDelta/2
	c.player.fov.RayEnd = c.player.direction + (HorizontalFOV*math.Pi)/360
}

func (c *Controller) mouseDirection(X1, Y1, X2, Y2 float32) float64 {
	return math.Mod(-math.Atan2(float64(X2-X1), float64(Y2-Y1)), 2*math.Pi) + math.Pi
}

func (c *Controller) mouseButtonEvent(event *sdl.MouseButtonEvent) bool {
	if event.State == 1 {
		if event.Button == 3 {
			c.player.position.X = float32(event.X)
			c.player.position.Y = float32(event.Y)
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
	c.shiftDown = event.Keysym.Mod&1 == 1
	if event.State == 1 {
		if event.Keysym.Scancode == sdl.SCANCODE_Q {
			c.Quit()
		} else if event.Keysym.Scancode == sdl.SCANCODE_P {
			c.walls = c.walls[:4]
			for i := 0; i < 5; i++ {
				c.walls = append(c.walls, &Wall{
					X1:    float32(rand.Int31n(c.arenaWidth / 2)),
					Y1:    float32(rand.Int31n(c.arenaHeight)),
					X2:    float32(rand.Int31n(c.arenaWidth / 2)),
					Y2:    float32(rand.Int31n(c.arenaHeight)),
					color: WallColors[i],
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

func (c *Controller) movePlayer(pt *sdl.FPoint) bool {
	for _, wall := range c.walls {
		if c.wallIntersect(c.player.position.X, c.player.position.Y, pt.X, pt.Y, wall.X1, wall.Y1, wall.X2, wall.Y2) {
			return false
		}
	}
	c.player.position = pt
	return true
}

func (c *Controller) forward() {
	X, Y, state := sdl.GetMouseState()
	if state == 1 && math.Abs(float64(X-int32(c.player.position.X))+math.Abs(float64(Y-int32(c.player.position.Y)))) <= 2 {
		c.movePlayer(&sdl.FPoint{X: float32(X), Y: float32(Y)})
	} else {
		c.movePlayer(c.translatePoint(c.player.position, c.player.direction, 2))
	}
}

func (c *Controller) backwards() {
	c.movePlayer(c.translatePoint(c.player.position, c.player.direction+math.Pi, StepSize))
}

func (c *Controller) rotate(clockwise bool) {
	if clockwise {
		c.player.direction += math.Pi / 90
	} else {
		c.player.direction -= math.Pi / 90
	}
	c.player.fov.RayStart = c.player.direction - (HorizontalFOV*math.Pi)/360 + c.player.fov.RayDelta/2
	c.player.fov.RayEnd = c.player.direction + (HorizontalFOV*math.Pi)/360
}

func (c *Controller) strafe(clockwise bool) {
	X, Y, state := sdl.GetMouseState()
	if state == 0 {
		offset := math.Pi / 2
		if clockwise {
			offset = -offset
		}
		c.movePlayer(c.translatePoint(c.player.position, c.player.direction+offset, 2))
	} else {
		r := c.lineLength(c.player.position.X, c.player.position.Y, float32(X), float32(Y))
		if r == 0 {
			return
		}
		t := float64(StepSize) / r
		angle := c.player.direction + t
		if !clockwise {
			angle = c.player.direction - t
		}
		o := math.Sin(angle) * r
		a := math.Cos(angle) * r
		pt := &sdl.FPoint{X: float32(X) - float32(o), Y: float32(Y) + float32(a)}
		if c.movePlayer(pt) {
			c.setFovAngle(X, Y)
			c.player.direction = angle
		}
	}
}

func (c *Controller) max(height float32, maxHeight float32) float32 {
	if height > maxHeight {
		height = maxHeight
	}
	return height

}
