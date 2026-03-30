package processing

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"strconv"
)

// EncodeWebP converts an image to WebP format using cwebp.
// It writes the image as PNG to a temp file, runs cwebp, and reads the result.
func EncodeWebP(img image.Image, quality int, cwebpPath string) ([]byte, error) {
	// Write image as PNG to temp file
	tmpIn, err := os.CreateTemp("", "mediastore-*.png")
	if err != nil {
		return nil, fmt.Errorf("creating temp input file: %w", err)
	}
	defer os.Remove(tmpIn.Name())

	if err := png.Encode(tmpIn, img); err != nil {
		tmpIn.Close()
		return nil, fmt.Errorf("encoding PNG: %w", err)
	}
	tmpIn.Close()

	// Create temp output path
	tmpOut, err := os.CreateTemp("", "mediastore-*.webp")
	if err != nil {
		return nil, fmt.Errorf("creating temp output file: %w", err)
	}
	tmpOutPath := tmpOut.Name()
	tmpOut.Close()
	defer os.Remove(tmpOutPath)

	// Run cwebp
	cmd := exec.Command(cwebpPath, "-q", strconv.Itoa(quality), tmpIn.Name(), "-o", tmpOutPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("cwebp failed: %w\noutput: %s", err, string(output))
	}

	// Read output
	data, err := os.ReadFile(tmpOutPath)
	if err != nil {
		return nil, fmt.Errorf("reading webp output: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("cwebp produced empty output")
	}

	return data, nil
}
