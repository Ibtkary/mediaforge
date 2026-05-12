package r2

import (
	"context"
	"image"
	"strings"
	"testing"
	"time"

	"github.com/Ibtkary/mediaforge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeEncoder struct{}

func (fakeEncoder) Encode(image.Image, int) ([]byte, error) { return []byte("webp"), nil }
func (fakeEncoder) Extension() string                       { return ".webp" }
func (fakeEncoder) ContentType() string                     { return "image/webp" }

func TestResolveEndpoint_DefaultFromAccountID(t *testing.T) {
	endpoint, err := resolveEndpoint(Config{AccountID: "abc123"})

	require.NoError(t, err)
	assert.Equal(t, "https://abc123.r2.cloudflarestorage.com", endpoint)
}

func TestResolveEndpoint_UsesOverrideAndTrimsSlash(t *testing.T) {
	endpoint, err := resolveEndpoint(Config{Endpoint: "https://r2.example.test/"})

	require.NoError(t, err)
	assert.Equal(t, "https://r2.example.test", endpoint)
}

func TestResolveEndpoint_RequiresAccountIDWhenEndpointMissing(t *testing.T) {
	_, err := resolveEndpoint(Config{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "account_id")
}

func TestValidateCredentials_MissingFields(t *testing.T) {
	err := validateCredentials(Config{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "access_key_id")
	assert.Contains(t, err.Error(), "secret_access_key")
	assert.Contains(t, err.Error(), "bucket")
}

func TestNewStorageAndSigner(t *testing.T) {
	storage, signer, err := NewStorageAndSigner(context.Background(), Config{
		Endpoint:        "https://r2.example.test",
		AccessKeyID:     "access-key",
		SecretAccessKey: "secret-key",
		Bucket:          "images",
	})

	require.NoError(t, err)
	assert.NotNil(t, storage)
	assert.NotNil(t, signer)
}

func TestNewClient_GeneratesPresignedURLWithoutNetwork(t *testing.T) {
	client, err := NewClient(context.Background(), Config{
		Endpoint:        "https://r2.example.test",
		AccessKeyID:     "access-key",
		SecretAccessKey: "secret-key",
		Bucket:          "images",
	}, mediaforge.WithEncoder(fakeEncoder{}))
	require.NoError(t, err)

	urls, err := client.GetSignedURL("stores/1/images/product.webp", []string{"thumb"}, time.Hour)

	require.NoError(t, err)
	assert.Contains(t, urls.Original, "X-Amz-Signature")
	assert.True(t, strings.Contains(urls.Original, "product.webp"))
	assert.Contains(t, urls.Variants["thumb"], "X-Amz-Signature")
}
