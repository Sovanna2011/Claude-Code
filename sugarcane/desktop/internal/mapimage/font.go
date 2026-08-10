package mapimage

import (
	"image"
	"image/color"
	"strings"
)

// A five by seven bitmap font, enough for the codes the map labels shapes with — letters, digits
// and the hyphen of FRM-01-Z1. It is here rather than taken from a font library so the map renders
// identically on the build machine and on Windows: the PNG a test checks is the picture the window
// shows, down to the pixel.
const (
	glyphWidth  = 5
	glyphHeight = 7
	glyphGap    = 1
)

var glyphs = map[rune][glyphHeight]string{
	'0': {".###.", "#...#", "#..##", "#.#.#", "##..#", "#...#", ".###."},
	'1': {"..#..", ".##..", "..#..", "..#..", "..#..", "..#..", ".###."},
	'2': {".###.", "#...#", "....#", "...#.", "..#..", ".#...", "#####"},
	'3': {"#####", "...#.", "..#..", "...#.", "....#", "#...#", ".###."},
	'4': {"...#.", "..##.", ".#.#.", "#..#.", "#####", "...#.", "...#."},
	'5': {"#####", "#....", "####.", "....#", "....#", "#...#", ".###."},
	'6': {"..##.", ".#...", "#....", "####.", "#...#", "#...#", ".###."},
	'7': {"#####", "....#", "...#.", "..#..", ".#...", ".#...", ".#..."},
	'8': {".###.", "#...#", "#...#", ".###.", "#...#", "#...#", ".###."},
	'9': {".###.", "#...#", "#...#", ".####", "....#", "...#.", ".##.."},
	'A': {".###.", "#...#", "#...#", "#####", "#...#", "#...#", "#...#"},
	'B': {"####.", "#...#", "#...#", "####.", "#...#", "#...#", "####."},
	'C': {".###.", "#...#", "#....", "#....", "#....", "#...#", ".###."},
	'D': {"###..", "#..#.", "#...#", "#...#", "#...#", "#..#.", "###.."},
	'E': {"#####", "#....", "#....", "####.", "#....", "#....", "#####"},
	'F': {"#####", "#....", "#....", "####.", "#....", "#....", "#...."},
	'G': {".###.", "#...#", "#....", "#.###", "#...#", "#...#", ".###."},
	'H': {"#...#", "#...#", "#...#", "#####", "#...#", "#...#", "#...#"},
	'I': {".###.", "..#..", "..#..", "..#..", "..#..", "..#..", ".###."},
	'J': {"..###", "...#.", "...#.", "...#.", "...#.", "#..#.", ".##.."},
	'K': {"#...#", "#..#.", "#.#..", "##...", "#.#..", "#..#.", "#...#"},
	'L': {"#....", "#....", "#....", "#....", "#....", "#....", "#####"},
	'M': {"#...#", "##.##", "#.#.#", "#...#", "#...#", "#...#", "#...#"},
	'N': {"#...#", "##..#", "#.#.#", "#..##", "#...#", "#...#", "#...#"},
	'O': {".###.", "#...#", "#...#", "#...#", "#...#", "#...#", ".###."},
	'P': {"####.", "#...#", "#...#", "####.", "#....", "#....", "#...."},
	'Q': {".###.", "#...#", "#...#", "#...#", "#.#.#", "#..#.", ".##.#"},
	'R': {"####.", "#...#", "#...#", "####.", "#.#..", "#..#.", "#...#"},
	'S': {".####", "#....", "#....", ".###.", "....#", "....#", "####."},
	'T': {"#####", "..#..", "..#..", "..#..", "..#..", "..#..", "..#.."},
	'U': {"#...#", "#...#", "#...#", "#...#", "#...#", "#...#", ".###."},
	'V': {"#...#", "#...#", "#...#", "#...#", "#...#", ".#.#.", "..#.."},
	'W': {"#...#", "#...#", "#...#", "#.#.#", "#.#.#", "##.##", "#...#"},
	'X': {"#...#", "#...#", ".#.#.", "..#..", ".#.#.", "#...#", "#...#"},
	'Y': {"#...#", "#...#", ".#.#.", "..#..", "..#..", "..#..", "..#.."},
	'Z': {"#####", "....#", "...#.", "..#..", ".#...", "#....", "#####"},
	'-': {".....", ".....", ".....", "#####", ".....", ".....", "....."},
	'.': {".....", ".....", ".....", ".....", ".....", ".....", "..#.."},
	'/': {"....#", "....#", "...#.", "..#..", ".#...", "#....", "#...."},
	' ': {".....", ".....", ".....", ".....", ".....", ".....", "....."},
}

// labelScale is the size of one glyph pixel in the supersampled image, so a code comes out seven
// pixels tall once the picture is scaled back down. That is small, and deliberately: a block on an
// estate-wide map is forty pixels across, and a label big enough to read comfortably covers the
// shape it names and its neighbours with it.
const labelScale = supersample

// labelWidth is how wide a code will be drawn, in supersampled pixels.
func labelWidth(text string) int {
	if text == "" {
		return 0
	}
	return len(text)*(glyphWidth+glyphGap)*labelScale - glyphGap*labelScale
}

// labelHeight is how tall a code will be drawn, in supersampled pixels.
func labelHeight() int { return glyphHeight * labelScale }

// drawLabel writes text centred on a point, with a halo so a code stays readable over a dark fill
// as well as a light one.
func drawLabel(img *image.RGBA, centreX, centreY int, text string, c color.RGBA) {
	text = strings.ToUpper(strings.TrimSpace(text))
	if text == "" {
		return
	}

	x := centreX - labelWidth(text)/2
	y := centreY - labelHeight()/2

	halo := color.RGBA{0xFF, 0xFF, 0xFF, 0xFF}
	if luminance(c) > 140 {
		halo = color.RGBA{0x20, 0x20, 0x20, 0xFF}
	}

	// The halo is the same text drawn one supersampled pixel out in four directions, which is
	// cheaper than a real outline and indistinguishable at this size. Four rather than eight: at
	// this glyph size the diagonals thicken the text into a smudge.
	for _, offset := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
		drawText(img, x+offset[0], y+offset[1], text, halo)
	}
	drawText(img, x, y, text, c)
}

func drawText(img *image.RGBA, x, y int, text string, c color.RGBA) {
	for _, ch := range text {
		glyph, ok := glyphs[ch]
		if !ok {
			glyph = glyphs[' ']
		}
		for row := 0; row < glyphHeight; row++ {
			line := glyph[row]
			for col := 0; col < glyphWidth && col < len(line); col++ {
				if line[col] != '#' {
					continue
				}
				block(img, x+col*labelScale, y+row*labelScale, labelScale, c)
			}
		}
		x += (glyphWidth + glyphGap) * labelScale
	}
}

func block(img *image.RGBA, x, y, size int, c color.RGBA) {
	bounds := img.Bounds()
	for dy := 0; dy < size; dy++ {
		for dx := 0; dx < size; dx++ {
			px, py := x+dx, y+dy
			if px >= bounds.Min.X && px < bounds.Max.X && py >= bounds.Min.Y && py < bounds.Max.Y {
				img.SetRGBA(px, py, c)
			}
		}
	}
}

func luminance(c color.RGBA) int {
	return int((299*uint32(c.R) + 587*uint32(c.G) + 114*uint32(c.B)) / 1000)
}
