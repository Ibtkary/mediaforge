package mediaforge

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"
	"time"
)

func skipIfNoCWebP(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("cwebp"); err != nil {
		t.Skip("cwebp not found, skipping test")
	}
}

func testClientWithServer(t *testing.T, handler http.Handler) (*Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	cfg := testConfig()
	cfg.MaxRetries = intPtr(0)
	cfg.UploadTimeout = 5 * time.Second
	cfg.DeleteTimeout = 5 * time.Second

	client, err := New(cfg)
	if err != nil {
		t.Fatalf("testClientWithServer: %v", err)
	}
	client.setStorageBaseURL(server.URL)
	return client, server
}

func createTestImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, red)
		}
	}
	return img
}

func createTestJPEG(width, height int) []byte {
	img := createTestImage(width, height)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		panic("failed to create test JPEG: " + err.Error())
	}
	return buf.Bytes()
}

func createTestPNG(width, height int) []byte {
	img := createTestImage(width, height)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic("failed to create test PNG: " + err.Error())
	}
	return buf.Bytes()
}
