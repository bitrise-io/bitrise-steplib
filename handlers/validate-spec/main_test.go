package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_validateSchema(t *testing.T) {
	schema, err := loadAsset("schema.json")
	require.NoError(t, err)

	spec, err := loadAsset("spec.json")
	require.NoError(t, err)

	require.NoError(t, validateSchema(spec, schema))
}
