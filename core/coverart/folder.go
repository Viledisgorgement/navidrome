package coverart

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var imageExts = map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true, ".bmp": true}

// WriteFolderCover saves data as cover<ext> in dir, the way Airsonic does.
// Any existing cover.* images are renamed to *.bak first so they are kept
// but no longer picked up as artwork. Returns the path written.
func WriteFolderCover(dir string, data []byte, ext string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	// Write to a temp file first, so a read-only or full disk fails before
	// anything existing is touched
	tmp, err := os.CreateTemp(dir, ".cover-*"+ext)
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if err := os.Chmod(tmpName, 0644); err != nil {
		return "", err
	}

	for _, e := range entries {
		name := e.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if e.IsDir() || !imageExts[ext] || !strings.EqualFold(strings.TrimSuffix(name, filepath.Ext(name)), "cover") {
			continue
		}
		old := filepath.Join(dir, name)
		if err := os.Rename(old, old+".bak"); err != nil {
			return "", fmt.Errorf("backing up %s: %w", name, err)
		}
	}

	target := filepath.Join(dir, "cover"+ext)
	if err := os.Rename(tmpName, target); err != nil {
		return "", err
	}
	return target, nil
}
