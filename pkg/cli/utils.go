package cli

import "github.com/urfave/cli/v3"

// joinFlags combines multiple flag slices into one
func joinFlags(flags ...[]cli.Flag) []cli.Flag {
	var result []cli.Flag
	for _, f := range flags {
		result = append(result, f...)
	}
	return result
}
