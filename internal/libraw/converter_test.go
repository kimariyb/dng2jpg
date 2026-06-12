package libraw

import (
	"os"
	"path/filepath"
	"testing"
)

func findDNG(t *testing.T) string {
	t.Helper()
	// Look for DNG files relative to project root
	candidates := []string{
		"../../dng",
		"../dng",
		"dng",
	}
	for _, dir := range candidates {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() && filepath.Ext(e.Name()) == ".dng" {
				return filepath.Join(dir, e.Name())
			}
		}
	}
	t.Skip("no DNG test files found")
	return ""
}

func TestConvert_JPEG(t *testing.T) {
	dng := findDNG(t)
	outDir := t.TempDir()
	outPath := filepath.Join(outDir, "test.jpg")

	opts := DefaultOptions()
	err := Convert(dng, outPath, opts)
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	// Verify output exists and is non-empty
	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("output file not created: %v", err)
	}
	if info.Size() == 0 {
		t.Error("output file is empty")
	}
	t.Logf("converted %s -> %s (%d bytes)", filepath.Base(dng), outPath, info.Size())
}

func TestConvert_NilOpts(t *testing.T) {
	dng := findDNG(t)
	outDir := t.TempDir()
	outPath := filepath.Join(outDir, "test.jpg")

	err := Convert(dng, outPath, nil)
	if err != nil {
		t.Fatalf("Convert with nil opts failed: %v", err)
	}
	info, _ := os.Stat(outPath)
	if info != nil {
		t.Logf("nil opts converted OK, size=%d", info.Size())
	}
}

func TestConvert_InvalidOpts(t *testing.T) {
	dng := findDNG(t)
	outDir := t.TempDir()
	outPath := filepath.Join(outDir, "test.jpg")

	opts := DefaultOptions()
	opts.JPEGQuality = 200
	err := Convert(dng, outPath, opts)
	if err == nil {
		t.Error("expected error for invalid quality")
	}
}
