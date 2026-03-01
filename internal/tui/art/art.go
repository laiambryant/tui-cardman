// Package art provides ASCII art rendering utilities for the TUI.
package art

import (
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const Logo = `
     ___   ___   ___   ___   _  _   ___   _  _
    / __| / _ \ | _ \ |   \ | \/ | / _ \ | \| |
   | (__ | (_| ||   / | |) || |\/|| (_| || .  |
    \___| \__._||_|_\ |___/ |_|  | \__._||_|\_|

       ~ Terminal UI Card Collection Manager ~
`

func brailleBitIdx(dx, dy int) uint {
	if dx == 0 {
		if dy < 3 {
			return uint(dy)
		}
		return 6
	}
	if dy < 3 {
		return uint(dy + 3)
	}
	return 7
}

func generatePattern(w, h int) []string {
	if w < 20 {
		w = 20
	}
	if h < 8 {
		h = 8
	}
	if w > 60 {
		w = 60
	}
	if h > 25 {
		h = 25
	}
	dotW := w * 2
	dotH := h * 4
	cx := float64(dotW) / 2.0
	cy := float64(dotH) / 2.0
	maxR := math.Min(cx, cy) * 0.85
	cardW := float64(dotW) * 0.9
	cardH := float64(dotH) * 0.9
	cardLeft := cx - cardW/2
	cardRight := cx + cardW/2
	cardTop := cy - cardH/2
	cardBottom := cy + cardH/2
	cornerR := math.Min(cardW, cardH) * 0.12
	grid := make([][]bool, dotH)
	for i := range grid {
		grid[i] = make([]bool, dotW)
	}
	for py := 0; py < dotH; py++ {
		for px := 0; px < dotW; px++ {
			fx := float64(px)
			fy := float64(py)
			dx := fx - cx
			dy := fy - cy
			dist := math.Sqrt(dx*dx + dy*dy)
			angle := math.Atan2(dy, dx)
			inCard := fx >= cardLeft && fx <= cardRight && fy >= cardTop && fy <= cardBottom
			if inCard {
				corners := [][2]float64{
					{cardLeft + cornerR, cardTop + cornerR},
					{cardRight - cornerR, cardTop + cornerR},
					{cardLeft + cornerR, cardBottom - cornerR},
					{cardRight - cornerR, cardBottom - cornerR},
				}
				for _, c := range corners {
					cdx := fx - c[0]
					cdy := fy - c[1]
					isCornerQuadrant := false
					if c[0] < cx && c[1] < cy && fx < c[0] && fy < c[1] {
						isCornerQuadrant = true
					} else if c[0] > cx && c[1] < cy && fx > c[0] && fy < c[1] {
						isCornerQuadrant = true
					} else if c[0] < cx && c[1] > cy && fx < c[0] && fy > c[1] {
						isCornerQuadrant = true
					} else if c[0] > cx && c[1] > cy && fx > c[0] && fy > c[1] {
						isCornerQuadrant = true
					}
					if isCornerQuadrant {
						cDist := math.Sqrt(cdx*cdx + cdy*cdy)
						if cDist > cornerR {
							inCard = false
						}
						break
					}
				}
			}
			if !inCard {
				continue
			}
			borderThick := 2.0
			nearEdge := fx-cardLeft < borderThick || cardRight-fx < borderThick ||
				fy-cardTop < borderThick || cardBottom-fy < borderThick
			if nearEdge {
				grid[py][px] = true
				continue
			}
			for _, c := range [][2]float64{
				{cardLeft + cornerR, cardTop + cornerR},
				{cardRight - cornerR, cardTop + cornerR},
				{cardLeft + cornerR, cardBottom - cornerR},
				{cardRight - cornerR, cardBottom - cornerR},
			} {
				cdx := fx - c[0]
				cdy := fy - c[1]
				cDist := math.Sqrt(cdx*cdx + cdy*cdy)
				if math.Abs(cDist-cornerR) < borderThick {
					isCornerQuadrant := false
					if c[0] < cx && c[1] < cy && fx < c[0] && fy < c[1] {
						isCornerQuadrant = true
					} else if c[0] > cx && c[1] < cy && fx > c[0] && fy < c[1] {
						isCornerQuadrant = true
					} else if c[0] < cx && c[1] > cy && fx < c[0] && fy > c[1] {
						isCornerQuadrant = true
					} else if c[0] > cx && c[1] > cy && fx > c[0] && fy > c[1] {
						isCornerQuadrant = true
					}
					if isCornerQuadrant {
						grid[py][px] = true
					}
				}
			}
			numRays := 8
			for i := 0; i < numRays; i++ {
				rayAngle := float64(i) * math.Pi / float64(numRays)
				angleDiff := math.Abs(angle - rayAngle)
				if angleDiff > math.Pi {
					angleDiff = 2*math.Pi - angleDiff
				}
				rayWidth := 0.06 - 0.02*(dist/maxR)
				if rayWidth < 0.02 {
					rayWidth = 0.02
				}
				if angleDiff < rayWidth && dist < maxR*0.75 && dist > maxR*0.15 {
					grid[py][px] = true
				}
			}
			adx := math.Abs(dx)
			ady := math.Abs(dy)
			diamondDist := adx + ady
			for _, r := range []float64{0.15, 0.35, 0.55, 0.75} {
				ringR := maxR * r
				if math.Abs(diamondDist-ringR) < 1.5 {
					grid[py][px] = true
				}
			}
			if diamondDist < maxR*0.08 {
				grid[py][px] = true
			}
		}
	}
	var result []string
	for row := 0; row < h; row++ {
		var b strings.Builder
		for col := 0; col < w; col++ {
			var code rune = 0x2800
			for dy := 0; dy < 4; dy++ {
				for dx := 0; dx < 2; dx++ {
					py := row*4 + dy
					px := col*2 + dx
					if py < dotH && px < dotW && grid[py][px] {
						code |= rune(1 << brailleBitIdx(dx, dy))
					}
				}
			}
			b.WriteRune(code)
		}
		result = append(result, b.String())
	}
	return result
}

func RenderLogo(width, height int, titleStyle, focusedStyle, blurredStyle lipgloss.Style) string {
	artW := width / 2
	artH := height * 45 / 100
	pattern := generatePattern(artW, artH)
	artStyle := lipgloss.NewStyle().Inherit(focusedStyle)
	var lines []string
	for _, line := range pattern {
		var b strings.Builder
		for _, ch := range line {
			if ch == 0x2800 {
				b.WriteRune(' ')
			} else {
				b.WriteRune(ch)
			}
		}
		lines = append(lines, artStyle.Render(b.String()))
	}
	patternArt := strings.Join(lines, "\n")
	cardmanText := titleStyle.Bold(true).Render("C A R D M A N")
	subtitle := blurredStyle.Render("~ Terminal UI Card Collection Manager ~")
	return lipgloss.JoinVertical(lipgloss.Center, patternArt, "", cardmanText, subtitle)
}
