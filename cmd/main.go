/*
 * Copyright (C) 2023 by Jason Figge
 */

package main

import (
	"fmt"

	"ray-tracing/internal"

	"us.figge/guilib/graphics"
)

const (
	screenWidth  = 1200
	screenHeight = 600
)

func main() {
	graphics.Open("Ray Caster", screenWidth, screenHeight, internal.NewController(screenWidth, screenHeight))
	fmt.Println("Game over")
}
