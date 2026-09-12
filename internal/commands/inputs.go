package commands

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/cristianoliveira/pxp/internal/cli"
	diff "github.com/cristianoliveira/pxp/internal/imagediff"
	"github.com/cristianoliveira/pxp/internal/imageio"
	"github.com/spf13/cobra"
)

type preparedImageInputs struct {
	referencePath string
	actualPath    string
	metadata      *diff.ImageInputs
	cleanup       func()
}

type exportMetadata struct {
	Version      int                   `json:"version"`
	NodeBounds   exportMetadataSize    `json:"nodeBounds"`
	ExportBounds exportMetadataSize    `json:"exportBounds"`
	LogicalCrop  *exportMetadataBounds `json:"logicalCrop"`
}

type exportMetadataBounds struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type exportMetadataSize struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

func addInputPreparationFlags(command *cobra.Command) {
	command.Flags().
		String("reference-crop", "", "crop reference before operation: x,y,width,height")
	command.Flags().
		String("reference-metadata", "", "apply logical crop from image export metadata JSON")
	command.Flags().String("actual-crop", "", "crop actual before operation: x,y,width,height")
}

func prepareCommandImageInputs(
	cmd *cobra.Command,
	referencePath, actualPath string,
) (preparedImageInputs, error) {
	referenceCrop, err := parseOptionalCrop(cmd, "reference-crop")
	if err != nil {
		return preparedImageInputs{}, cli.NewUsageError(err)
	}
	actualCrop, err := parseOptionalCrop(cmd, "actual-crop")
	if err != nil {
		return preparedImageInputs{}, cli.NewUsageError(err)
	}

	referenceMetadataPath, _ := cmd.Flags().GetString("reference-metadata")
	if referenceCrop != nil && referenceMetadataPath != "" {
		return preparedImageInputs{}, cli.NewUsageError(
			fmt.Errorf("--reference-crop and --reference-metadata cannot be used together"),
		)
	}
	referenceMetadata, err := loadExportMetadata(referenceMetadataPath)
	if err != nil {
		return preparedImageInputs{}, err
	}

	return prepareImageInputs(
		referencePath,
		actualPath,
		referenceCrop,
		actualCrop,
		referenceMetadata,
	)
}

func prepareImageInputs(
	referencePath, actualPath string,
	referenceCrop, actualCrop *diff.Bounds,
	referenceMetadata *exportMetadata,
) (preparedImageInputs, error) {
	referenceWidth, referenceHeight, err := imageio.PNGDimensions(referencePath)
	if err != nil {
		return preparedImageInputs{}, fmt.Errorf("decode reference: %w", err)
	}
	actualWidth, actualHeight, err := imageio.PNGDimensions(actualPath)
	if err != nil {
		return preparedImageInputs{}, fmt.Errorf("decode actual: %w", err)
	}

	if referenceMetadata != nil {
		if int(referenceMetadata.ExportBounds.Width) != referenceWidth ||
			int(referenceMetadata.ExportBounds.Height) != referenceHeight {
			return preparedImageInputs{}, cli.NewUsageError(
				fmt.Errorf(
					"--reference-metadata export bounds %gx%g do not match reference image %dx%d",
					referenceMetadata.ExportBounds.Width,
					referenceMetadata.ExportBounds.Height,
					referenceWidth,
					referenceHeight,
				),
			)
		}
		referenceCrop = cropFromExportMetadata(*referenceMetadata)
	}

	if referenceCrop == nil && actualCrop == nil {
		return preparedImageInputs{
			referencePath: referencePath,
			actualPath:    actualPath,
			cleanup:       func() {},
		}, nil
	}

	if err := validateCrop(referenceCrop, referenceWidth, referenceHeight); err != nil {
		return preparedImageInputs{}, cli.NewUsageError(
			fmt.Errorf("invalid --reference-crop: %w", err),
		)
	}
	if err := validateCrop(actualCrop, actualWidth, actualHeight); err != nil {
		return preparedImageInputs{}, cli.NewUsageError(
			fmt.Errorf("invalid --actual-crop: %w", err),
		)
	}

	referenceCompareWidth, referenceCompareHeight := croppedDimensions(
		referenceWidth,
		referenceHeight,
		referenceCrop,
	)
	actualCompareWidth, actualCompareHeight := croppedDimensions(
		actualWidth,
		actualHeight,
		actualCrop,
	)
	if referenceCompareWidth != actualCompareWidth ||
		referenceCompareHeight != actualCompareHeight {
		return preparedImageInputs{}, cli.NewUsageError(
			fmt.Errorf(
				"cropped image dimensions differ: reference is %dx%d, actual is %dx%d",
				referenceCompareWidth,
				referenceCompareHeight,
				actualCompareWidth,
				actualCompareHeight,
			),
		)
	}

	tempDir, err := os.MkdirTemp("", "pxp-crops-*")
	if err != nil {
		return preparedImageInputs{}, err
	}
	cleanup := func() { _ = os.RemoveAll(tempDir) }

	metadata := &diff.ImageInputs{
		Reference: diff.ImageInput{
			Width:  referenceWidth,
			Height: referenceHeight,
			Crop:   referenceCrop,
		},
		Actual: diff.ImageInput{
			Width:  actualWidth,
			Height: actualHeight,
			Crop:   actualCrop,
		},
	}
	prepared := preparedImageInputs{
		referencePath: referencePath,
		actualPath:    actualPath,
		metadata:      metadata,
		cleanup:       cleanup,
	}

	if referenceCrop != nil {
		prepared.referencePath = filepath.Join(tempDir, "reference.png")
		//nolint:lll // keep this expression together
		if err := imageio.WriteCroppedPNG(referencePath, prepared.referencePath, *referenceCrop); err != nil {
			cleanup()
			return preparedImageInputs{}, fmt.Errorf("invalid --reference-crop: %w", err)
		}
	}
	if actualCrop != nil {
		prepared.actualPath = filepath.Join(tempDir, "actual.png")
		if err := imageio.WriteCroppedPNG(actualPath, prepared.actualPath, *actualCrop); err != nil {
			cleanup()
			return preparedImageInputs{}, fmt.Errorf("invalid --actual-crop: %w", err)
		}
	}

	return prepared, nil
}

func parseOptionalCrop(cmd *cobra.Command, flagName string) (*diff.Bounds, error) {
	value, _ := cmd.Flags().GetString(flagName)
	crop, err := parseImageRegion(value)
	if err != nil {
		return nil, fmt.Errorf("invalid --%s: %w", flagName, err)
	}
	if crop != nil && (crop.Width < 1 || crop.Height < 1) {
		return nil, fmt.Errorf("invalid --%s: width and height must be positive", flagName)
	}
	return crop, nil
}

func loadExportMetadata(path string) (*exportMetadata, error) {
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var metadata exportMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}
	if metadata.Version != 1 {
		return nil, fmt.Errorf("unsupported --reference-metadata version %d", metadata.Version)
	}
	return &metadata, nil
}

func cropFromExportMetadata(metadata exportMetadata) *diff.Bounds {
	if metadata.LogicalCrop != nil {
		crop := &diff.Bounds{
			X:      int(math.Round(metadata.LogicalCrop.X)),
			Y:      int(math.Round(metadata.LogicalCrop.Y)),
			Width:  int(math.Round(metadata.LogicalCrop.Width)),
			Height: int(math.Round(metadata.LogicalCrop.Height)),
		}
		if crop.X >= 0 && crop.Y >= 0 &&
			crop.X+crop.Width <= int(math.Round(metadata.ExportBounds.Width)) &&
			crop.Y+crop.Height <= int(math.Round(metadata.ExportBounds.Height)) {
			return crop
		}
	}
	return &diff.Bounds{
		X:      int(math.Floor((metadata.ExportBounds.Width - metadata.NodeBounds.Width) / 2)),
		Y:      int(math.Floor((metadata.ExportBounds.Height - metadata.NodeBounds.Height) / 2)),
		Width:  int(math.Round(metadata.NodeBounds.Width)),
		Height: int(math.Round(metadata.NodeBounds.Height)),
	}
}

func inputBounds(bounds diff.Bounds, inputs *diff.ImageInputs) *diff.InputBounds {
	if inputs == nil {
		return nil
	}
	return &diff.InputBounds{
		Reference: boundsWithCropOrigin(bounds, inputs.Reference.Crop),
		Actual:    boundsWithCropOrigin(bounds, inputs.Actual.Crop),
	}
}

func boundsWithCropOrigin(bounds diff.Bounds, crop *diff.Bounds) diff.Bounds {
	if crop == nil {
		return bounds
	}
	bounds.X += crop.X
	bounds.Y += crop.Y
	return bounds
}

func croppedDimensions(width, height int, crop *diff.Bounds) (int, int) {
	if crop == nil {
		return width, height
	}
	return crop.Width, crop.Height
}

func validateCrop(crop *diff.Bounds, width, height int) error {
	if crop == nil {
		return nil
	}
	if crop.X < 0 || crop.Y < 0 || crop.Width <= 0 || crop.Height <= 0 ||
		crop.X+crop.Width > width ||
		crop.Y+crop.Height > height {
		return fmt.Errorf(
			"crop %d,%d,%d,%d is outside image bounds %dx%d",
			crop.X,
			crop.Y,
			crop.Width,
			crop.Height,
			width,
			height,
		)
	}
	return nil
}
