package mordye

import (
	"fmt"
	"math"
	"strconv"
)

// rgb is an sRGB color with channels in [0, 1].
type rgb struct{ R, G, B float64 }

// lch is a color in OKLCH: perceptual lightness (0-1), chroma, and hue in
// degrees. OKLCH is where Mordye does its color math, because equal steps in
// it look equal: a generated yellow at the palette's lightness reads as light
// as the palette's own colors, which plain RGB or HSL doesn't promise.
// https://bottosson.github.io/posts/oklab/
type lch struct{ L, C, H float64 }

func parseHex(s string) (rgb, error) {
	if len(s) != 7 || s[0] != '#' {
		return rgb{}, fmt.Errorf("color %q: want #rrggbb", s)
	}
	v, err := strconv.ParseUint(s[1:], 16, 32)
	if err != nil {
		return rgb{}, fmt.Errorf("color %q: want #rrggbb", s)
	}
	return rgb{
		R: float64(v>>16&0xff) / 255,
		G: float64(v>>8&0xff) / 255,
		B: float64(v&0xff) / 255,
	}, nil
}

func (c rgb) hex() string {
	ch := func(x float64) uint8 { return uint8(math.Round(math.Max(0, math.Min(1, x)) * 255)) }
	return fmt.Sprintf("#%02x%02x%02x", ch(c.R), ch(c.G), ch(c.B))
}

func toLinear(c float64) float64 {
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func fromLinear(c float64) float64 {
	if c <= 0.0031308 {
		return 12.92 * c
	}
	return 1.055*math.Pow(c, 1/2.4) - 0.055
}

func (c rgb) lch() lch {
	r, g, b := toLinear(c.R), toLinear(c.G), toLinear(c.B)
	l := math.Cbrt(0.4122214708*r + 0.5363325363*g + 0.0514459929*b)
	m := math.Cbrt(0.2119034982*r + 0.6806995451*g + 0.1073969566*b)
	s := math.Cbrt(0.0883024619*r + 0.2817188376*g + 0.6299787005*b)
	L := 0.2104542553*l + 0.7936177850*m - 0.0040720468*s
	A := 1.9779984951*l - 2.4285922050*m + 0.4505937099*s
	B := 0.0259040371*l + 0.7827717662*m - 0.8086757660*s
	return lch{L: L, C: math.Hypot(A, B), H: math.Mod(math.Atan2(B, A)*180/math.Pi+360, 360)}
}

// linear converts to linear sRGB, which may fall outside [0, 1] when the
// color is out of gamut.
func (c lch) linear() (float64, float64, float64) {
	a := c.C * math.Cos(c.H*math.Pi/180)
	b := c.C * math.Sin(c.H*math.Pi/180)
	l := math.Pow(c.L+0.3963377774*a+0.2158037573*b, 3)
	m := math.Pow(c.L-0.1055613458*a-0.0638541728*b, 3)
	s := math.Pow(c.L-0.0894841775*a-1.2914855480*b, 3)
	return 4.0767416621*l - 3.3077115913*m + 0.2309699292*s,
		-1.2684380046*l + 2.6097574011*m - 0.3413193965*s,
		-0.0041960863*l - 0.7034186147*m + 1.7076147010*s
}

// hex converts to sRGB hex. A color outside sRGB loses chroma until it fits,
// keeping its lightness and hue, which is what keeps a generated color in
// family with the palette.
func (c lch) hex() string {
	const tolerance = 1e-4
	inGamut := func(x float64) bool { return x >= -tolerance && x <= 1+tolerance }
	for {
		r, g, b := c.linear()
		if (inGamut(r) && inGamut(g) && inGamut(b)) || c.C <= 0 {
			return rgb{fromLinear(math.Max(0, r)), fromLinear(math.Max(0, g)), fromLinear(math.Max(0, b))}.hex()
		}
		c.C = math.Max(0, c.C-0.002)
	}
}

// Chroma returns a #rrggbb color's OKLCH chroma: 0 for a pure gray, about
// 0.3 for the most saturated sRGB colors. Adapters use it to tell a tinted
// color from a plain one.
func Chroma(hex string) (float64, error) {
	c, err := parseHex(hex)
	if err != nil {
		return 0, err
	}
	return c.lch().C, nil
}

// hueGap is the shortest distance between two hues, in degrees.
func hueGap(a, b float64) float64 {
	return math.Abs(math.Mod(a-b+540, 360) - 180)
}
