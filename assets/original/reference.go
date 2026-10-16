// Package reference provides original executables to development comparisons.
// The running game imports the parent assets package and does not link this one.
package reference

import "embed"

//go:embed game.bin ikplus.hunk
var programs embed.FS

// Read returns an unchanged source executable for offline extraction or tests.
func Read(name string) ([]byte, error) { return programs.ReadFile(name) }
