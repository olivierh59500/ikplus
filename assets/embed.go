// Package assets embeds graphics, animation tables and PCM samples recovered
// reproducibly from the supplied Amiga disk. Runtime code never needs the ADF.
package assets

import (
	"embed"
	"io/fs"
	"sort"
)

// original contains native assets produced by tools/extract_*.py. Source
// executables are supplied by the separate development-only reference package.
//
//go:embed original/*.json original/*.png original/*.indices original/*.wav original/*.pcm original/random-*.bin original/fighter-shadows.bin
var original embed.FS

// Read returns a recovered asset using its path relative to this package,
// for example "original/background.png" or "original/animations.json".
func Read(path string) ([]byte, error) {
	return original.ReadFile(path)
}

// Names lists the embedded files in lexical order for diagnostics and tools.
func Names() ([]string, error) {
	var names []string
	err := fs.WalkDir(original, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			names = append(names, path)
		}
		return nil
	})
	sort.Strings(names)
	return names, err
}
