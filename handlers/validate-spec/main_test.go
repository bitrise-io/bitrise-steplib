package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_validateSchema(t *testing.T) {
	require.NoError(t, validateSchema(spec))
}

var spec []byte

func init() {
	var err error
	spec, err = loadAsset("spec.json")
	if err != nil {
		panic(err)
	}
}
