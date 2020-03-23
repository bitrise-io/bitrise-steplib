package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_validateSchema_validSpec(t *testing.T) {
	require.NoError(t, validateSchema(spec, schema))
}

func Test_validateSchema_invalidSpec(t *testing.T) {
	err := validateSchema(invalidSpec, schema)
	fmt.Println(err)
	require.Error(t, err)
	require.Equal(t, `/format_version: "1" regexp pattrn ^\d+.\d+.\d+$ mismatch on string: 1
/steplib_source: "file://." regexp pattrn ^https://github.com/bitrise-io/bitrise-steplib.git$ mismatch on string: file://.`, err.Error())
}

var spec = []byte(`{
    "format_version": "1.0.0",
    "generated_at_timestamp": 1584720375,
	"steplib_source": "https://github.com/bitrise-io/bitrise-steplib.git"
}`)

var invalidSpec = []byte(`{
    "format_version": "1",
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
            "pattern": "^\\d+.\\d+.\\d+$",
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
            "pattern": "^https://github.com/bitrise-io/bitrise-steplib.git$",
            "examples": [
                "https://github.com/bitrise-io/bitrise-steplib.git"
            ]
		}
	}
}`)
