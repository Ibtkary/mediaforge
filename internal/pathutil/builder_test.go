package pathutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeTenantID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "valid simple", input: "tenant1", want: "tenant1"},
		{name: "valid with hyphen", input: "my-tenant", want: "my-tenant"},
		{name: "valid with underscore", input: "my_tenant", want: "my_tenant"},
		{name: "valid alphanumeric", input: "Tenant123", want: "Tenant123"},
		{name: "valid UUID-like", input: "550e8400-e29b-41d4-a716-446655440000", want: "550e8400-e29b-41d4-a716-446655440000"},
		{name: "empty", input: "", wantErr: true},
		{name: "path traversal", input: "tenant/../etc", wantErr: true},
		{name: "forward slash", input: "tenant/sub", wantErr: true},
		{name: "backslash", input: "tenant\\sub", wantErr: true},
		{name: "spaces", input: "my tenant", wantErr: true},
		{name: "dot dot only", input: "..", wantErr: true},
		{name: "special chars", input: "tenant@#$", wantErr: true},
		{name: "unicode", input: "تينانت", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SanitizeTenantID(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildStoragePath(t *testing.T) {
	tests := []struct {
		name     string
		tenant   string
		filename string
		want     string
	}{
		{
			name:     "normal",
			tenant:   "t1",
			filename: "abc123_def456.webp",
			want:     "stores/t1/images/abc123_def456.webp",
		},
		{
			name:     "long tenant",
			tenant:   "my-long-tenant-id",
			filename: "image.webp",
			want:     "stores/my-long-tenant-id/images/image.webp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, BuildStoragePath(tt.tenant, tt.filename))
		})
	}
}

func TestBuildOriginalFilename(t *testing.T) {
	tests := []struct {
		name       string
		hashPrefix string
		uuidPrefix string
		want       string
	}{
		{
			name:       "normal",
			hashPrefix: "e3b0c44298fc",
			uuidPrefix: "a1b2c3d4",
			want:       "e3b0c44298fc_a1b2c3d4.webp",
		},
		{
			name:       "short prefixes",
			hashPrefix: "abc",
			uuidPrefix: "123",
			want:       "abc_123.webp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, BuildOriginalFilename(tt.hashPrefix, tt.uuidPrefix))
		})
	}
}

func TestBuildVariantFilename(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		variant  string
		want     string
	}{
		{
			name:    "thumb variant",
			base:    "e3b0c44298fc_a1b2c3d4.webp",
			variant: "thumb",
			want:    "e3b0c44298fc_a1b2c3d4_thumb.webp",
		},
		{
			name:    "medium variant",
			base:    "e3b0c44298fc_a1b2c3d4.webp",
			variant: "medium",
			want:    "e3b0c44298fc_a1b2c3d4_medium.webp",
		},
		{
			name:    "large variant",
			base:    "abc_def.webp",
			variant: "large",
			want:    "abc_def_large.webp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, BuildVariantFilename(tt.base, tt.variant))
		})
	}
}

func TestBuildCDNURL(t *testing.T) {
	tests := []struct {
		name    string
		cdnBase string
		path    string
		want    string
	}{
		{
			name:    "no trailing slash",
			cdnBase: "https://cdn.nafees.com",
			path:    "stores/t1/images/abc.webp",
			want:    "https://cdn.nafees.com/stores/t1/images/abc.webp",
		},
		{
			name:    "with trailing slash",
			cdnBase: "https://cdn.nafees.com/",
			path:    "stores/t1/images/abc.webp",
			want:    "https://cdn.nafees.com/stores/t1/images/abc.webp",
		},
		{
			name:    "path with leading slash",
			cdnBase: "https://cdn.nafees.com",
			path:    "/stores/t1/images/abc.webp",
			want:    "https://cdn.nafees.com/stores/t1/images/abc.webp",
		},
		{
			name:    "both slashes",
			cdnBase: "https://cdn.nafees.com/",
			path:    "/stores/t1/images/abc.webp",
			want:    "https://cdn.nafees.com/stores/t1/images/abc.webp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, BuildCDNURL(tt.cdnBase, tt.path))
		})
	}
}

func TestExtractVariantPaths(t *testing.T) {
	tests := []struct {
		name     string
		basePath string
		variants []string
		want     []string
	}{
		{
			name:     "standard variants",
			basePath: "stores/t1/images/abc_def.webp",
			variants: []string{"thumb", "medium", "large"},
			want: []string{
				"stores/t1/images/abc_def_thumb.webp",
				"stores/t1/images/abc_def_medium.webp",
				"stores/t1/images/abc_def_large.webp",
			},
		},
		{
			name:     "single variant",
			basePath: "stores/t1/images/abc_def.webp",
			variants: []string{"thumb"},
			want: []string{
				"stores/t1/images/abc_def_thumb.webp",
			},
		},
		{
			name:     "no variants",
			basePath: "stores/t1/images/abc_def.webp",
			variants: []string{},
			want:     []string{},
		},
		{
			name:     "filename only",
			basePath: "abc_def.webp",
			variants: []string{"thumb"},
			want:     []string{"abc_def_thumb.webp"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractVariantPaths(tt.basePath, tt.variants)
			assert.Equal(t, tt.want, got)
		})
	}
}
