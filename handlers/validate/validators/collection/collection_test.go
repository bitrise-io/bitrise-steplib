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
			name:                "2 unsupported version + version NOT included",
			unsupportedVersions: []string{"3.2.1", "1.2.1"},
			want:                false,
		},
		{
			name:                "2 unsupported version + version included",
			unsupportedVersions: []string{"3.2.1", "1.2.3"},
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
