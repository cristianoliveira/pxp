package imagediff

import (
	"fmt"
	"image"
	"sort"
)

const AnatomyQuery = "foreground-elements"

type AnatomyOptions struct {
	Threshold  uint8
	Background *[4]uint8
	Group      int
	MinPixels  int
}

type AnatomyElement struct {
	Bounds        Bounds  `json:"bounds"`
	Pixels        int     `json:"pixels"`
	Density       float64 `json:"density"`
	DominantColor string  `json:"dominantColor"`
}

type AnatomyResult struct {
	Width      int              `json:"width"`
	Height     int              `json:"height"`
	Background string           `json:"background"`
	Threshold  uint8            `json:"threshold"`
	Group      int              `json:"group"`
	MinPixels  int              `json:"minPixels"`
	Query      string           `json:"query"`
	Total      int              `json:"total"`
	Returned   int              `json:"returned"`
	Truncated  bool             `json:"truncated"`
	Hint       string           `json:"hint,omitempty"`
	Message    string           `json:"message,omitempty"`
	Elements   []AnatomyElement `json:"elements"`
}

type anatomyComponent struct {
	bounds Bounds
	pixels int
	colors map[[4]uint8]int
}

func DiscoverAnatomy(img *image.NRGBA, options AnatomyOptions) (AnatomyResult, error) {
	if img == nil {
		return AnatomyResult{}, fmt.Errorf("image is required")
	}
	if options.Group < 0 {
		return AnatomyResult{}, fmt.Errorf("group must be non-negative")
	}
	if options.MinPixels < 1 {
		return AnatomyResult{}, fmt.Errorf("min-pixels must be greater than zero")
	}

	bounds := img.Bounds()
	background := detectBackground(img, options.Background)
	foreground := foregroundPixels(img, bounds, background, options.Threshold)
	components := findAnatomyComponents(img, foreground, bounds, options.MinPixels)
	if options.Group > 0 {
		components = groupAnatomyComponents(components, options.Group)
	}
	sort.SliceStable(components, func(i, j int) bool {
		if components[i].bounds.Y != components[j].bounds.Y {
			return components[i].bounds.Y < components[j].bounds.Y
		}
		if components[i].bounds.X != components[j].bounds.X {
			return components[i].bounds.X < components[j].bounds.X
		}
		if components[i].bounds.Height != components[j].bounds.Height {
			return components[i].bounds.Height < components[j].bounds.Height
		}
		return components[i].bounds.Width < components[j].bounds.Width
	})

	result := AnatomyResult{
		Width: bounds.Dx(), Height: bounds.Dy(),
		Background: formatPixelColor(background[:]), Threshold: options.Threshold,
		Group: options.Group, MinPixels: options.MinPixels,
		Query: AnatomyQuery, Total: len(components), Returned: len(components),
		Elements: make([]AnatomyElement, len(components)),
	}
	for index, component := range components {
		result.Elements[index] = anatomyElement(component)
	}
	if len(result.Elements) == 0 {
		result.Message = fmt.Sprintf(
			"no %s found against background %s",
			AnatomyQuery,
			result.Background,
		)
	}
	return result, nil
}

func detectBackground(img *image.NRGBA, explicit *[4]uint8) [4]uint8 {
	if explicit != nil {
		return *explicit
	}
	counts := make(map[[4]uint8]int)
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			counts[pixelAt(img, x, y)]++
		}
	}
	candidates := make([][4]uint8, 0, len(counts))
	for pixel := range counts {
		candidates = append(candidates, pixel)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if counts[candidates[i]] != counts[candidates[j]] {
			return counts[candidates[i]] > counts[candidates[j]]
		}
		return pixelLess(candidates[i], candidates[j])
	})
	if len(candidates) == 0 {
		return [4]uint8{0, 0, 0, 0}
	}
	return candidates[0]
}

func foregroundPixels(
	img *image.NRGBA, bounds image.Rectangle, background [4]uint8, threshold uint8,
) []bool {
	foreground := make([]bool, bounds.Dx()*bounds.Dy())
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			pixel := pixelAt(img, x+bounds.Min.X, y+bounds.Min.Y)
			if background[3] == 255 && pixel[3] < 128 {
				continue
			}
			foreground[y*bounds.Dx()+x] = maxColorDelta(pixel, background) > threshold
		}
	}
	return foreground
}

func findAnatomyComponents(
	img *image.NRGBA, foreground []bool, bounds image.Rectangle, minPixels int,
) []anatomyComponent {
	width, height := bounds.Dx(), bounds.Dy()
	visited := make([]bool, len(foreground))
	components := make([]anatomyComponent, 0)
	for start := range foreground {
		if !foreground[start] || visited[start] {
			continue
		}
		queue := []int{start}
		visited[start] = true
		component := anatomyComponent{bounds: Bounds{X: width, Y: height}, colors: make(map[[4]uint8]int)}
		maxX, maxY := -1, -1
		for len(queue) > 0 {
			index := queue[0]
			queue = queue[1:]
			x, y := index%width, index/width
			component.pixels++
			component.colors[pixelAt(img, x+bounds.Min.X, y+bounds.Min.Y)]++
			component.bounds.X = min(component.bounds.X, x+bounds.Min.X)
			component.bounds.Y = min(component.bounds.Y, y+bounds.Min.Y)
			maxX, maxY = max(maxX, x+bounds.Min.X), max(maxY, y+bounds.Min.Y)
			for _, point := range [][2]int{{x - 1, y}, {x + 1, y}, {x, y - 1}, {x, y + 1}} {
				nx, ny := point[0], point[1]
				if nx < 0 || nx >= width || ny < 0 || ny >= height {
					continue
				}
				next := ny*width + nx
				if foreground[next] && !visited[next] {
					visited[next] = true
					queue = append(queue, next)
				}
			}
		}
		if component.pixels < minPixels {
			continue
		}
		component.bounds.Width = maxX - component.bounds.X + 1
		component.bounds.Height = maxY - component.bounds.Y + 1
		components = append(components, component)
	}
	return components
}

func groupAnatomyComponents(components []anatomyComponent, gap int) []anatomyComponent {
	grouped := append([]anatomyComponent(nil), components...)
	for {
		merged := false
		for i := 0; i < len(grouped); i++ {
			for j := i + 1; j < len(grouped); j++ {
				if !anatomyBoundsWithinGap(grouped[i].bounds, grouped[j].bounds, gap) {
					continue
				}
				grouped[i] = mergeAnatomyComponents(grouped[i], grouped[j])
				grouped = append(grouped[:j], grouped[j+1:]...)
				merged = true
				break
			}
			if merged {
				break
			}
		}
		if !merged {
			return grouped
		}
	}
}

func anatomyBoundsWithinGap(first, second Bounds, gap int) bool {
	horizontal := max(0, max(first.X, second.X)-min(first.X+first.Width, second.X+second.Width))
	vertical := max(0, max(first.Y, second.Y)-min(first.Y+first.Height, second.Y+second.Height))
	return horizontal <= gap && vertical <= gap
}

func mergeAnatomyComponents(first, second anatomyComponent) anatomyComponent {
	merged := anatomyComponent{
		bounds: Bounds{
			X: min(first.bounds.X, second.bounds.X), Y: min(first.bounds.Y, second.bounds.Y),
			Width: max(
				first.bounds.X+first.bounds.Width, second.bounds.X+second.bounds.Width,
			) - min(first.bounds.X, second.bounds.X),
			Height: max(
				first.bounds.Y+first.bounds.Height, second.bounds.Y+second.bounds.Height,
			) - min(first.bounds.Y, second.bounds.Y),
		},
		pixels: first.pixels + second.pixels,
		colors: make(map[[4]uint8]int, len(first.colors)+len(second.colors)),
	}
	for pixel, count := range first.colors {
		merged.colors[pixel] += count
	}
	for pixel, count := range second.colors {
		merged.colors[pixel] += count
	}
	return merged
}

func anatomyElement(component anatomyComponent) AnatomyElement {
	dominant := dominantAnatomyColor(component.colors)
	return AnatomyElement{
		Bounds: component.bounds, Pixels: component.pixels,
		Density: float64(component.pixels) /
			float64(component.bounds.Width*component.bounds.Height),
		DominantColor: formatPixelColor(dominant[:]),
	}
}

func dominantAnatomyColor(colors map[[4]uint8]int) [4]uint8 {
	var result [4]uint8
	best := -1
	for pixel, count := range colors {
		if count > best || (count == best && pixelLess(pixel, result)) {
			result, best = pixel, count
		}
	}
	return result
}

func pixelAt(img *image.NRGBA, x, y int) [4]uint8 {
	pixel := img.NRGBAAt(x, y)
	return [4]uint8{pixel.R, pixel.G, pixel.B, pixel.A}
}

func maxColorDelta(first, second [4]uint8) uint8 {
	return max(
		absDiff(first[0], second[0]), absDiff(first[1], second[1]),
		absDiff(first[2], second[2]), absDiff(first[3], second[3]),
	)
}

func pixelLess(first, second [4]uint8) bool {
	for index := range first {
		if first[index] != second[index] {
			return first[index] < second[index]
		}
	}
	return false
}
