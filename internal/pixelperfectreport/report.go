package pixelperfectreport

import (
	"bytes"
	"encoding/base64"
	"html/template"
	"os"

	"github.com/cristianoliveira/pxp/internal/imagediff"
	"github.com/cristianoliveira/pxp/internal/output"
)

type Input struct {
	ReferencePath       string
	ActualPath          string
	MaskPath            string
	OverlayPath         string
	Threshold           uint8
	PerceptualThreshold float64
	ComparedRegion      *imagediff.Bounds
	Result              imagediff.ImageComparison
}

type view struct {
	Input
	ReferenceDataURL template.URL
	ActualDataURL    template.URL
	MaskDataURL      template.URL
	OverlayDataURL   template.URL
}

func Write(path string, input Input) error {
	html, err := Render(input)
	if err != nil {
		return err
	}
	return output.WriteFile(path, html, 0o600)
}

func Render(input Input) ([]byte, error) {
	view := view{Input: input}
	var err error
	view.ReferenceDataURL, err = imageDataURL(input.ReferencePath)
	if err != nil {
		return nil, err
	}
	view.ActualDataURL, err = imageDataURL(input.ActualPath)
	if err != nil {
		return nil, err
	}
	view.MaskDataURL, err = imageDataURL(input.MaskPath)
	if err != nil {
		return nil, err
	}
	if input.OverlayPath != "" {
		view.OverlayDataURL, err = imageDataURL(input.OverlayPath)
		if err != nil {
			return nil, err
		}
	}
	var output bytes.Buffer
	if err := reportTemplate.Execute(&output, view); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func imageDataURL(path string) (template.URL, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return template.URL("data:image/png;base64," + base64.StdEncoding.EncodeToString(data)), nil
}

var reportTemplate = template.Must(template.New("report").Parse(`<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><title>Pixel Perfect Report</title></head>
<body>
<h1>Pixel Perfect Report</h1>
<section><h2>Run configuration and provenance</h2>
<ul><li>Reference: {{.ReferencePath}}</li><li>Actual: {{.ActualPath}}</li><li>Mask: {{.MaskPath}}</li>{{if .OverlayPath}}<li>Overlay: {{.OverlayPath}}</li>{{end}}<li>Threshold: {{.Threshold}}</li><li>Perceptual threshold: {{.PerceptualThreshold}}</li>{{if .ComparedRegion}}<li>Compared region: {{.ComparedRegion.X}},{{.ComparedRegion.Y}},{{.ComparedRegion.Width}},{{.ComparedRegion.Height}}</li>{{end}}</ul>
{{if .Result.Inputs}}<h3>Input transforms</h3><ul>{{if .Result.Inputs.Reference.Crop}}<li>Reference crop: {{.Result.Inputs.Reference.Crop.X}},{{.Result.Inputs.Reference.Crop.Y}},{{.Result.Inputs.Reference.Crop.Width}},{{.Result.Inputs.Reference.Crop.Height}}</li>{{end}}{{if .Result.Inputs.Actual.Crop}}<li>Actual crop: {{.Result.Inputs.Actual.Crop.X}},{{.Result.Inputs.Actual.Crop.Y}},{{.Result.Inputs.Actual.Crop.Width}},{{.Result.Inputs.Actual.Crop.Height}}</li>{{end}}</ul>{{end}}
</section>
<section><h2>Global metrics</h2>
<ul><li>Changed pixels: {{.Result.ChangedPixels}}</li><li>Changed ratio: {{.Result.ChangedRatio}}</li><li>RMSE: {{.Result.RMSE}}</li><li>Perceptual changed ratio: {{.Result.PerceptualChangedRatio}}</li></ul>
</section>
<section><h2>Artifacts</h2>
<h3>Reference</h3><img alt="Reference" src="{{.ReferenceDataURL}}">
<h3>Actual</h3><img alt="Actual" src="{{.ActualDataURL}}">
<h3>Mask</h3><img alt="Mask" src="{{.MaskDataURL}}">
{{if .OverlayDataURL}}<h3>Overlay</h3><img alt="Overlay" src="{{.OverlayDataURL}}">{{end}}
</section>
<section><h2>Ranked deterministic regions</h2>
{{if .Result.Regions}}<table><thead><tr><th>#</th><th>Cropped bounds</th><th>Reference input bounds</th><th>Actual input bounds</th><th>Changed pixels</th><th>Changed ratio</th><th>RMSE</th><th>Classification</th></tr></thead><tbody>{{range $index, $region := .Result.Regions}}<tr><td>{{$index}}</td><td>{{$region.Bounds.X}},{{$region.Bounds.Y}},{{$region.Bounds.Width}},{{$region.Bounds.Height}}</td><td>{{if $region.InputBounds}}{{$region.InputBounds.Reference.X}},{{$region.InputBounds.Reference.Y}},{{$region.InputBounds.Reference.Width}},{{$region.InputBounds.Reference.Height}}{{end}}</td><td>{{if $region.InputBounds}}{{$region.InputBounds.Actual.X}},{{$region.InputBounds.Actual.Y}},{{$region.InputBounds.Actual.Width}},{{$region.InputBounds.Actual.Height}}{{end}}</td><td>{{$region.ChangedPixels}}</td><td>{{$region.ChangedRatio}}</td><td>{{$region.RMSE}}</td><td>{{$region.Classification}}</td></tr>{{end}}</tbody></table>{{else}}<p>No changed regions.</p>{{end}}
</section>
{{if .Result.SuggestedOffset}}<section><h2>Suggested offset</h2><p>x={{.Result.SuggestedOffset.X}} y={{.Result.SuggestedOffset.Y}} rmse={{.Result.SuggestedOffset.RMSE}}</p></section>{{end}}
{{if .Result.MovedRegions}}<section><h2>Advisory region movements</h2><p>These candidates do not align images or alter validation.</p><ul>{{range .Result.MovedRegions}}<li>bounds={{.Bounds.X}},{{.Bounds.Y}},{{.Bounds.Width}},{{.Bounds.Height}} dx={{.DX}} dy={{.DY}} confidence={{.Confidence}}</li>{{end}}</ul></section>{{end}}
</body></html>
`))
