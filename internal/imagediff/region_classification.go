package imagediff

func ClassifyImageRegion(metrics RegionMetrics) string {
	if metrics.ChangedPixels == 0 {
		return "none"
	}
	if metrics.ChangedRatio >= 0.6 && dominantPairRatio(metrics) >= 0.7 &&
		metrics.EdgeRMSE < metrics.RMSE*0.5 {
		return "solid-fill"
	}
	if metrics.EdgeRMSE >= metrics.RMSE*0.7 {
		return "geometry"
	}
	if metrics.ChangedRatio < 0.35 {
		return "sparse-raster"
	}
	return "mixed"
}

func dominantPairRatio(metrics RegionMetrics) float64 {
	if metrics.ChangedPixels == 0 || len(metrics.DominantColorPairs) == 0 {
		return 0
	}
	return float64(metrics.DominantColorPairs[0].Pixels) / float64(metrics.ChangedPixels)
}
