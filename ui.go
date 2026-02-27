package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var (
	colorBtnBg     = color.RGBA{50, 50, 65, 255}
	colorBtnActive = color.RGBA{70, 70, 110, 255}
	colorBtnBorder = color.RGBA{100, 100, 130, 255}
)

// drawButton draws a labeled rectangular button. If highlighted is true, uses a brighter bg.
func drawButton(screen *ebiten.Image, x, y, w, h int, label string, highlighted bool) {
	bg := colorBtnBg
	if highlighted {
		bg = colorBtnActive
	}
	fx, fy, fw, fh := float32(x), float32(y), float32(w), float32(h)

	// Fill
	var path vector.Path
	path.MoveTo(fx, fy)
	path.LineTo(fx+fw, fy)
	path.LineTo(fx+fw, fy+fh)
	path.LineTo(fx, fy+fh)
	path.Close()
	vector.FillPath(screen, &path, nil, &vector.DrawPathOptions{
		ColorScale: colorScaleFrom(bg),
	})

	// Border
	var border vector.Path
	border.MoveTo(fx, fy)
	border.LineTo(fx+fw, fy)
	border.LineTo(fx+fw, fy+fh)
	border.LineTo(fx, fy+fh)
	border.Close()
	vector.StrokePath(screen, &border, &vector.StrokeOptions{Width: 1}, &vector.DrawPathOptions{
		ColorScale: colorScaleFrom(colorBtnBorder),
	})

	// Centered label
	charW := 6
	lx := x + (w-len(label)*charW)/2
	ly := y + (h-16)/2
	ebitenutil.DebugPrintAt(screen, label, lx, ly)
}

// btnClicked returns true if left mouse (or touch) was just pressed inside the rect.
func btnClicked(x, y, w, h int) bool {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		if mx >= x && mx < x+w && my >= y && my < y+h {
			return true
		}
	}
	for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
		tx, ty := ebiten.TouchPosition(id)
		if tx >= x && tx < x+w && ty >= y && ty < y+h {
			return true
		}
	}
	return false
}

// btnRightClicked returns true if right mouse was just pressed inside the rect.
func btnRightClicked(x, y, w, h int) bool {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		mx, my := ebiten.CursorPosition()
		return mx >= x && mx < x+w && my >= y && my < y+h
	}
	return false
}

// cycleIdx advances or retreats an index in a list of length n, wrapping around.
func cycleIdx(idx *int, n int, forward bool) {
	if forward {
		*idx = (*idx + 1) % n
	} else {
		*idx = (*idx + n - 1) % n
	}
}
