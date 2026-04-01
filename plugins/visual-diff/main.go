package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"

	spectra "github.com/spectra-browser/spectra/sdk/go"
)

type CompareParams struct {
	spectra.BrowserOptions
	URL1      string  `json:"url1"`
	URL2      string  `json:"url2"`
	Threshold float64 `json:"threshold"`
}

func main() {
	p := spectra.NewPlugin("visual-diff")

	p.Handle("compare", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		var p CompareParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		if p.URL1 == "" || p.URL2 == "" {
			return nil, fmt.Errorf("url1 and url2 are required")
		}
		if p.Threshold == 0 {
			p.Threshold = 0.1
		}

		s1, err := spectra.OpenPage(p.URL1, p.BrowserOptions)
		if err != nil {
			return nil, fmt.Errorf("open url1: %w", err)
		}
		img1, err := s1.Page.Screenshot(false, nil)
		s1.Close()
		if err != nil {
			return nil, fmt.Errorf("screenshot url1: %w", err)
		}

		s2, err := spectra.OpenPage(p.URL2, p.BrowserOptions)
		if err != nil {
			return nil, fmt.Errorf("open url2: %w", err)
		}
		img2, err := s2.Page.Screenshot(false, nil)
		s2.Close()
		if err != nil {
			return nil, fmt.Errorf("screenshot url2: %w", err)
		}

		diffPNG, diffPercent, diffPixels, err := diffImages(img1, img2, p.Threshold)
		if err != nil {
			return nil, fmt.Errorf("diff: %w", err)
		}

		return map[string]interface{}{
			"diff_image":   base64.StdEncoding.EncodeToString(diffPNG),
			"image1":       base64.StdEncoding.EncodeToString(img1),
			"image2":       base64.StdEncoding.EncodeToString(img2),
			"diff_percent": math.Round(diffPercent*100) / 100,
			"diff_pixels":  diffPixels,
			"threshold":    p.Threshold,
			"match":        diffPercent < 1.0,
			"width":        p.BrowserOptions.ViewportWidth(),
			"height":       p.BrowserOptions.ViewportHeight(),
		}, nil
	})

	p.Run()
}

func diffImages(img1Data, img2Data []byte, threshold float64) ([]byte, float64, int, error) {
	i1, err := png.Decode(bytes.NewReader(img1Data))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("decode img1: %w", err)
	}
	i2, err := png.Decode(bytes.NewReader(img2Data))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("decode img2: %w", err)
	}

	w, h := minInt(i1.Bounds().Dx(), i2.Bounds().Dx()), minInt(i1.Bounds().Dy(), i2.Bounds().Dy())
	diff := image.NewRGBA(image.Rect(0, 0, w, h))
	diffPixels := 0

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r1, g1, b1, _ := i1.At(x, y).RGBA()
			r2, g2, b2, _ := i2.At(x, y).RGBA()
			delta := (math.Abs(float64(r1)-float64(r2)) + math.Abs(float64(g1)-float64(g2)) + math.Abs(float64(b1)-float64(b2))) / 3 / 65535.0
			if delta > threshold {
				diffPixels++
				diff.Set(x, y, color.RGBA{R: 255, A: 180})
			} else {
				r, g, b, _ := i1.At(x, y).RGBA()
				gray := uint8((r + g + b) / 3 / 256)
				diff.Set(x, y, color.RGBA{R: gray, G: gray, B: gray, A: 255})
			}
		}
	}

	var buf bytes.Buffer
	png.Encode(&buf, diff)
	return buf.Bytes(), float64(diffPixels) / float64(w*h) * 100, diffPixels, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
