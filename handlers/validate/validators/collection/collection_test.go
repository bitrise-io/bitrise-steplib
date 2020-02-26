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
		v                   *version.Version
		unsupportedVersions []string
		want                bool
	}{
		{
			name:                "no unsupported versions",
			v:                   v,
			unsupportedVersions: nil,
			want:                false,
		},
		{
			name:                "one supported version",
			v:                   v,
			unsupportedVersions: []string{"3.2.1"},
			want:                false,
		},
		{
			name:                "multiple supported versions",
			v:                   v,
			unsupportedVersions: []string{"3.2.1", "4.2.1", "1.2.1"},
			want:                false,
		},
		{
			name:                "one unsupported version",
			v:                   v,
			unsupportedVersions: []string{"1.2.3"},
			want:                true,
		},
		{
			name:                "one unsupported version from multiple versions",
			v:                   v,
			unsupportedVersions: []string{"3.2.1", "1.2.3", "1.2.1"},
			want:                true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isUnsupportedVersion(tt.v, tt.unsupportedVersions...); got != tt.want {
				t.Errorf("isUnsupportedVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}
