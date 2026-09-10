package pixelperfectcmd

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"strconv"
	"strings"

	"github.com/cristianoliveira/pxp/internal/cli"
	diff "github.com/cristianoliveira/pxp/internal/imagediff"
	outputpkg "github.com/cristianoliveira/pxp/internal/output"
	"github.com/spf13/cobra"
)

type probeOutput struct {
	Total     int                `json:"total"`
	Returned  int                `json:"returned"`
	Truncated bool               `json:"truncated"`
	Hint      string             `json:"hint,omitempty"`
	Points    []probePointOutput `json:"points"`
	Inputs    *diff.ImageInputs  `json:"inputs,omitempty"`
}

type probePointOutput struct {
	Point      probePoint       `json:"point"`
	Reference  probeColor       `json:"reference"`
	Actual     probeColor       `json:"actual"`
	Delta      probeDelta       `json:"delta"`
	InputPoint *probeInputPoint `json:"inputPoint,omitempty"`
}

type probeInputPoint struct {
	Reference probePoint `json:"reference"`
	Actual    probePoint `json:"actual"`
}

type probePoint struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type probeColor struct {
	RGBA [4]uint8 `json:"rgba"`
	Hex  string   `json:"hex"`
}

type probeDelta struct {
	R int `json:"r"`
	G int `json:"g"`
	B int `json:"b"`
	A int `json:"a"`
}

func newProbeCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "probe <reference.png> <actual.png>",
		Short: "Inspect colors at one pixel in two equal-sized PNGs",
		Args:  requireImagePair("probe", "pxp probe reference.png actual.png --at 12,24"),
		Example: `  pxp probe reference.png actual.png --at 12,24
  pxp probe reference.png actual.png --from 0,20 --to 100,20
  pxp probe reference.png actual.png --at 12,24 --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			limit, err := inspectionResultLimit(cmd)
			if err != nil {
				return cli.NewUsageError(err)
			}
			points, err := probePointsFromFlags(cmd)
			if err != nil {
				return cli.NewUsageError(err)
			}
			format, err := tabularFormat(cmd)
			if err != nil {
				return cli.NewUsageError(err)
			}
			inputs, err := prepareCommandImageInputs(cmd, args[0], args[1])
			if err != nil {
				return err
			}
			defer inputs.cleanup()
			output, err := probeImages(inputs.referencePath, inputs.actualPath, points)
			if err != nil {
				return err
			}
			output.Inputs = inputs.metadata
			output.Total = len(output.Points)
			if !limit.full && len(output.Points) > limit.maximum {
				output.Points = output.Points[:limit.maximum]
				output.Truncated = true
				output.Hint = cli.FullHint(cmd, args)
			}
			output.Returned = len(output.Points)
			for index := range output.Points {
				output.Points[index].InputPoint = inputPoint(output.Points[index].Point, inputs.metadata)
			}
			if format == outputpkg.FormatJSON {
				return writeJSON(cmd, output)
			}
			return writeProbeCSV(cmd, output)
		},
	}
	command.Flags().StringArray("at", nil, "pixel coordinate to inspect: x,y in comparison/cropped coordinates; repeat for multiple points")
	command.Flags().String("from", "", "inclusive line start: x,y in comparison/cropped coordinates")
	command.Flags().String("to", "", "inclusive line end: x,y in comparison/cropped coordinates")
	command.Flags().Int("step", 1, "sample every Nth point along --from/--to line")
	command.Flags().Int("radius", 0, "include square pixel neighborhood around every selected point")
	addInspectionLimitFlags(command, "points")
	addTabularFormatFlag(command)
	addInputPreparationFlags(command)
	return command
}

func inputPoint(point probePoint, inputs *diff.ImageInputs) *probeInputPoint {
	if inputs == nil {
		return nil
	}
	return &probeInputPoint{
		Reference: pointWithCropOrigin(point, inputs.Reference.Crop),
		Actual:    pointWithCropOrigin(point, inputs.Actual.Crop),
	}
}

func pointWithCropOrigin(point probePoint, crop *diff.Bounds) probePoint {
	if crop == nil {
		return point
	}
	return probePoint{X: point.X + crop.X, Y: point.Y + crop.Y}
}

func probePointsFromFlags(command *cobra.Command) ([]probePoint, error) {
	step, _ := command.Flags().GetInt("step")
	if step < 1 {
		return nil, fmt.Errorf("--step must be positive")
	}
	radius, _ := command.Flags().GetInt("radius")
	if radius < 0 {
		return nil, fmt.Errorf("--radius must be non-negative")
	}
	values, _ := command.Flags().GetStringArray("at")
	points, err := parseProbePoints(values)
	if err != nil {
		return nil, err
	}
	fromValue, _ := command.Flags().GetString("from")
	toValue, _ := command.Flags().GetString("to")
	if (fromValue == "") != (toValue == "") {
		return nil, fmt.Errorf("--from and --to must be provided together")
	}
	if fromValue != "" {
		from, err := parseProbePoint(fromValue)
		if err != nil {
			return nil, fmt.Errorf("invalid --from: %w", err)
		}
		to, err := parseProbePoint(toValue)
		if err != nil {
			return nil, fmt.Errorf("invalid --to: %w", err)
		}
		points = append(points, steppedProbeLine(from, to, step)...)
	}
	if len(points) == 0 {
		return nil, fmt.Errorf("provide at least one --at point or --from/--to line")
	}
	return expandProbeRadius(points, radius), nil
}

func parseProbePoints(values []string) ([]probePoint, error) {
	points := make([]probePoint, 0, len(values))
	for _, value := range values {
		point, err := parseProbePoint(value)
		if err != nil {
			return nil, err
		}
		points = append(points, point)
	}
	return points, nil
}

func steppedProbeLine(from, to probePoint, step int) []probePoint {
	line := rasterProbeLine(from, to)
	points := make([]probePoint, 0, (len(line)+step-1)/step+1)
	for index := 0; index < len(line); index += step {
		points = append(points, line[index])
	}
	if points[len(points)-1] != to {
		points = append(points, to)
	}
	return points
}

func rasterProbeLine(from, to probePoint) []probePoint {
	x, y := from.X, from.Y
	dx, dy := absInt(to.X-from.X), -absInt(to.Y-from.Y)
	stepX, stepY := -1, -1
	if x < to.X {
		stepX = 1
	}
	if y < to.Y {
		stepY = 1
	}
	err := dx + dy
	points := make([]probePoint, 0, max(dx, -dy)+1)
	for {
		points = append(points, probePoint{X: x, Y: y})
		if x == to.X && y == to.Y {
			return points
		}
		twiceError := 2 * err
		if twiceError >= dy {
			err += dy
			x += stepX
		}
		if twiceError <= dx {
			err += dx
			y += stepY
		}
	}
}

func expandProbeRadius(points []probePoint, radius int) []probePoint {
	seen := make(map[probePoint]struct{})
	expanded := make([]probePoint, 0, len(points)*(radius*2+1)*(radius*2+1))
	for _, point := range points {
		for y := max(0, point.Y-radius); y <= point.Y+radius; y++ {
			for x := max(0, point.X-radius); x <= point.X+radius; x++ {
				candidate := probePoint{X: x, Y: y}
				if _, exists := seen[candidate]; exists {
					continue
				}
				seen[candidate] = struct{}{}
				expanded = append(expanded, candidate)
			}
		}
	}
	return expanded
}

func parseProbePoint(value string) (probePoint, error) {
	parts := strings.Split(value, ",")
	if len(parts) != 2 {
		return probePoint{}, fmt.Errorf("--at must be x,y")
	}
	x, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return probePoint{}, fmt.Errorf("--at x must be an integer")
	}
	y, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return probePoint{}, fmt.Errorf("--at y must be an integer")
	}
	if x < 0 || y < 0 {
		return probePoint{}, fmt.Errorf("--at coordinates must be non-negative")
	}
	return probePoint{X: x, Y: y}, nil
}

func probeImages(referencePath, actualPath string, points []probePoint) (probeOutput, error) {
	referenceWidth, referenceHeight, err := diff.PNGDimensions(referencePath)
	if err != nil {
		return probeOutput{}, fmt.Errorf("decode reference: %w", err)
	}
	actualWidth, actualHeight, err := diff.PNGDimensions(actualPath)
	if err != nil {
		return probeOutput{}, fmt.Errorf("decode actual: %w", err)
	}
	if referenceWidth != actualWidth || referenceHeight != actualHeight {
		return probeOutput{}, fmt.Errorf("image dimensions differ: reference is %dx%d, actual is %dx%d", referenceWidth, referenceHeight, actualWidth, actualHeight)
	}
	output := probeOutput{Points: make([]probePointOutput, 0, len(points))}
	for _, point := range points {
		if point.X >= referenceWidth || point.Y >= referenceHeight {
			return probeOutput{}, cli.NewUsageError(fmt.Errorf("--at point %d,%d is outside image bounds %dx%d", point.X, point.Y, referenceWidth, referenceHeight))
		}
		referenceColor, err := probePNGColor(referencePath, point)
		if err != nil {
			return probeOutput{}, fmt.Errorf("decode reference: %w", err)
		}
		actualColor, err := probePNGColor(actualPath, point)
		if err != nil {
			return probeOutput{}, fmt.Errorf("decode actual: %w", err)
		}
		output.Points = append(output.Points, probePointOutput{
			Point:     point,
			Reference: referenceColor,
			Actual:    actualColor,
			Delta: probeDelta{
				R: int(referenceColor.RGBA[0]) - int(actualColor.RGBA[0]),
				G: int(referenceColor.RGBA[1]) - int(actualColor.RGBA[1]),
				B: int(referenceColor.RGBA[2]) - int(actualColor.RGBA[2]),
				A: int(referenceColor.RGBA[3]) - int(actualColor.RGBA[3]),
			},
		})
	}
	return output, nil
}

func probePNGColor(path string, point probePoint) (probeColor, error) {
	file, err := os.Open(path)
	if err != nil {
		return probeColor{}, err
	}
	defer func() { _ = file.Close() }()
	image, err := png.Decode(file)
	if err != nil {
		return probeColor{}, err
	}
	return colorFromImage(image, point), nil
}

func colorFromImage(image image.Image, point probePoint) probeColor {
	r, g, b, a := image.At(point.X, point.Y).RGBA()
	color := probeColor{RGBA: [4]uint8{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}}
	color.Hex = fmt.Sprintf("#%02X%02X%02X", color.RGBA[0], color.RGBA[1], color.RGBA[2])
	return color
}
