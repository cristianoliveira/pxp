package pixelperfectcmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/cristianoliveira/pxp/internal/cli"
	"github.com/spf13/cobra"
)

type comparisonProfile struct {
	Version         int  `json:"version"`
	SuggestOffset   *int `json:"suggestOffset,omitempty"`
	RegionGap       *int `json:"regionGap,omitempty"`
	MinRegionPixels *int `json:"minRegionPixels,omitempty"`
}

type resolvedValue[T any] struct {
	Value  T      `json:"value"`
	Source string `json:"source"`
}

type resolvedComparisonOptions struct {
	SuggestOffset   resolvedValue[int] `json:"suggestOffset"`
	RegionGap       resolvedValue[int] `json:"regionGap"`
	MinRegionPixels resolvedValue[int] `json:"minRegionPixels"`
}

type comparisonConfiguration struct {
	Profile  string                    `json:"profile"`
	Resolved resolvedComparisonOptions `json:"resolved"`
}

func applyComparisonProfile(command *cobra.Command) (*comparisonConfiguration, error) {
	path, _ := command.Flags().GetString("profile")
	if path == "" {
		return nil, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read --profile: %w", err)
	}
	defer func() { _ = file.Close() }()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var profile comparisonProfile
	if err := decoder.Decode(&profile); err != nil {
		return nil, cli.NewUsageError(fmt.Errorf("decode --profile: %w", err))
	}
	if profile.Version != 1 {
		return nil, cli.NewUsageError(fmt.Errorf("unsupported --profile version %d", profile.Version))
	}

	sources := map[string]string{}
	for _, option := range []struct {
		flag  string
		value *int
	}{
		{flag: "suggest-offset", value: profile.SuggestOffset},
		{flag: "region-gap", value: profile.RegionGap},
		{flag: "min-region-pixels", value: profile.MinRegionPixels},
	} {
		if command.Flags().Changed(option.flag) {
			sources[option.flag] = "flag"
			continue
		}
		if option.value == nil {
			sources[option.flag] = "default"
			continue
		}
		if err := command.Flags().Set(option.flag, strconv.Itoa(*option.value)); err != nil {
			return nil, cli.NewUsageError(err)
		}
		sources[option.flag] = "profile"
	}

	suggestOffset, _ := command.Flags().GetInt("suggest-offset")
	regionGap, _ := command.Flags().GetInt("region-gap")
	minRegionPixels, _ := command.Flags().GetInt("min-region-pixels")
	return &comparisonConfiguration{
		Profile: path,
		Resolved: resolvedComparisonOptions{
			SuggestOffset:   resolvedValue[int]{Value: suggestOffset, Source: sources["suggest-offset"]},
			RegionGap:       resolvedValue[int]{Value: regionGap, Source: sources["region-gap"]},
			MinRegionPixels: resolvedValue[int]{Value: minRegionPixels, Source: sources["min-region-pixels"]},
		},
	}, nil
}
