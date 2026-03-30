package processing

import (
	"image"

	xdraw "golang.org/x/image/draw"
)

// ResizeImage downscales img to fit within maxWidth×maxHeight while preserving aspect ratio.
// If the image already fits, it is copied to NRGBA without upscaling.
// Returns *image.NRGBA.
func ResizeImage(img image.Image, maxWidth, maxHeight int) *image.NRGBA {
	srcBounds := img.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	// If already fits, copy to NRGBA
	if srcW <= maxWidth && srcH <= maxHeight {
		dst := image.NewNRGBA(image.Rect(0, 0, srcW, srcH))
		xdraw.Copy(dst, image.Point{}, img, srcBounds, xdraw.Src, nil)
		return dst
	}

	// Calculate scale maintaining aspect ratio
	scaleW := float64(maxWidth) / float64(srcW)
	scaleH := float64(maxHeight) / float64(srcH)
	scale := scaleW
	if scaleH < scale {
		scale = scaleH
	}

	newW := int(float64(srcW) * scale)
	newH := int(float64(srcH) * scale)
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}

	dst := image.NewNRGBA(image.Rect(0, 0, newW, newH))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, srcBounds, xdraw.Src, nil)

	return dst
}
