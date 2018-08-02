package tools

import (
	"strings"

	"github.com/bitrise-io/go-utils/command"
)

// ExportEnvironmentWithEnvman ...
func ExportEnvironmentWithEnvman(key, value string) error {
	cmd := command.New("envman", "add", "--key", key)
	cmd.SetStdin(strings.NewReader(value))
	return cmd.Run()
}
