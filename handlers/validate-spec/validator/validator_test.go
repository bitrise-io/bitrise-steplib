package validator

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/bitrise-io/go-utils/sliceutil"
	"github.com/bitrise-io/stepman/models"
	"github.com/stretchr/testify/require"
)

func Test_validateTags(t *testing.T) {
	tests := []struct {
		name    string
		stepLib models.StepCollectionModel
		want    []string
	}{
		{
			name: "project type tag is optional",
			stepLib: models.StepCollectionModel{
				Steps: models.StepHash{
					"script": models.StepGroupModel{
						LatestVersionNumber: "1.0.0",
						Versions: map[string]models.StepModel{
							"1.0.0": models.StepModel{
								TypeTags:        []string{"utility"},
								ProjectTypeTags: []string{},
							},
						},
					},
				},
			},
			want: []string{},
		},
		{
			name: "type tag is required",
			stepLib: models.StepCollectionModel{
				Steps: models.StepHash{
					"script": models.StepGroupModel{
						LatestVersionNumber: "1.0.0",
						Versions: map[string]models.StepModel{
							"1.0.0": models.StepModel{
								TypeTags: []string{},
							},
						},
					},
				},
			},
			want: []string{"script (version 1.0.0) has no type_tags"},
		},
		{
			name: "checks only latest versions",
			stepLib: models.StepCollectionModel{
				Steps: models.StepHash{
					"script": models.StepGroupModel{
						LatestVersionNumber: "1.0.0",
						Versions: map[string]models.StepModel{
							"0.9.0": models.StepModel{
								TypeTags: []string{},
							},
							"1.0.0": models.StepModel{
								TypeTags: []string{"utility"},
							},
						},
					},
				},
			},
			want: []string{},
		},
		{
			name: "type tags are restricted to a given list",
			stepLib: models.StepCollectionModel{
				Steps: models.StepHash{
					"script": models.StepGroupModel{
						LatestVersionNumber: "1.0.0",
						Versions: map[string]models.StepModel{
							"1.0.0": models.StepModel{
								TypeTags: []string{"xcode"},
							},
						},
					},
				},
			},
			want: []string{"script (version 1.0.0) has invalid type_tag: xcode"},
		},
		{
			name: "project type tags are restricted to a given list",
			stepLib: models.StepCollectionModel{
				Steps: models.StepHash{
					"script": models.StepGroupModel{
						LatestVersionNumber: "1.0.0",
						Versions: map[string]models.StepModel{
							"1.0.0": models.StepModel{
								TypeTags:        []string{"utility"},
								ProjectTypeTags: []string{"xcode"},
							},
						},
					},
				},
			},
			want: []string{"script (version 1.0.0) has invalid project_type_tag: xcode"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateTags(tt.stepLib)
			require.NoError(t, err)

			require.Equal(t, len(tt.want), len(got))
			for _, actual := range got {
				require.True(t, sliceutil.IsStringInSlice(actual.Error(), tt.want), fmt.Sprintf("%s should be in %s", actual.Error(), tt.want))
			}
		})
	}
}

func TestValidateSchemaValidSpec(t *testing.T) {
	v, err := New(bytes.NewReader(spec))
	require.NoError(t, err)

	issues, err := v.ValidateSchema(bytes.NewReader(schema))
	require.NoError(t, err)
	require.Equal(t, 0, len(issues))
}

func TestValidateSchemaInvalidSpec(t *testing.T) {
	expected := []string{
		"/format_version: \"1.0.0\\n0.0.0\" regexp pattrn ^\\d+\\.\\d+\\.\\d+$ mismatch on string: 1.0.0\n0.0.0",
		"/steplib_source: \"file://.\" regexp pattrn ^https://github\\.com/bitrise-io/bitrise-steplib\\.git$ mismatch on string: file://.",
	}
	v, err := New(bytes.NewReader(invalidSpec))
	require.NoError(t, err)

	issues, err := v.ValidateSchema(bytes.NewReader(schema))
	require.NoError(t, err)
	require.Equal(t, len(expected), len(issues))
	for _, actual := range issues {
		require.True(t, sliceutil.IsStringInSlice(actual.Error(), expected))
	}
}

var spec = []byte(`{
    "format_version": "0.0.0",
    "generated_at_timestamp": 1584720375,
	"steplib_source": "https://github.com/bitrise-io/bitrise-steplib.git"
}`)

var invalidSpec = []byte(`{
    "format_version": "1.0.0\n0.0.0",
    "generated_at_timestamp": 1584720375,
	"steplib_source": "file://."
}`)

var schema = []byte(`{
    "$schema": "http://json-schema.org/draft-07/schema",
    "type": "object",
    "required": [
        "format_version",
        "generated_at_timestamp",
        "steplib_source"
    ],
	"additionalProperties": false,
    "properties": {
        "format_version": {
            "type": "string",
            "pattern": "^\\d+\\.\\d+\\.\\d+$",
            "examples": [
                "1.0.0"
            ]
        },
        "generated_at_timestamp": {
            "type": "integer",
            "examples": [
                1584720375
            ]
        },
        "steplib_source": {
            "type": "string",
            "pattern": "^https://github\\.com/bitrise-io/bitrise-steplib\\.git$",
            "examples": [
                "https://github.com/bitrise-io/bitrise-steplib.git"
            ]
		}
	}
}`)
