package collection

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hashicorp/go-version"
)

func Test_isUnsupportedVersion(t *testing.T) {
	v, err := version.NewVersion("1.2.3")
	require.NoError(t, err)

	tests := []struct {
		name                string
		unsupportedVersions []string
		want                bool
	}{
		{
			name:                "no unsupported versions",
			unsupportedVersions: nil,
			want:                false,
		},
		{
			name:                "one supported version",
			unsupportedVersions: []string{"3.2.1"},
			want:                false,
		},
		{
			name:                "multiple supported versions",
			unsupportedVersions: []string{"3.2.1", "4.2.1", "1.2.1"},
			want:                false,
		},
		{
			name:                "one unsupported version",
			unsupportedVersions: []string{"1.2.3"},
			want:                true,
		},
		{
			name:                "one unsupported version from multiple versions",
			unsupportedVersions: []string{"3.2.1", "1.2.3", "1.2.1"},
			want:                true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isUnsupportedVersion(v, tt.unsupportedVersions...); got != tt.want {
				t.Errorf("isUnsupportedVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}
