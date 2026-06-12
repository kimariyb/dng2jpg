package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDeriveOutput_JPG(t *testing.T) {
	got := deriveOutput("/path/to/photo.dng", ".jpg")
	want := "photo.jpg"
	if got != want {
		t.Errorf("deriveOutput = %q, want %q", got, want)
	}
}

func TestDeriveOutput_PNG(t *testing.T) {
	got := deriveOutput("/path/to/photo.dng", ".png")
	want := "photo.png"
	if got != want {
		t.Errorf("deriveOutput = %q, want %q", got, want)
	}
}

func TestDeriveOutput_CurrentDir(t *testing.T) {
	got := deriveOutput("photo.dng", ".jpg")
	want := "photo.jpg"
	if got != want {
		t.Errorf("deriveOutput = %q, want %q", got, want)
	}
}

func TestUniquePath_NoConflict(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "test.jpg")
	got := uniquePath(p)
	if got != p {
		t.Errorf("expected %q, got %q", p, got)
	}
}

func TestUniquePath_Conflict(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "test.jpg")
	os.WriteFile(p, []byte("x"), 0644)
	got := uniquePath(p)
	expected := filepath.Join(dir, "test_1.jpg")
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestUniquePath_MultipleConflicts(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "test.jpg")
	os.WriteFile(p, []byte("x"), 0644)
	os.WriteFile(filepath.Join(dir, "test_1.jpg"), []byte("x"), 0644)
	os.WriteFile(filepath.Join(dir, "test_2.jpg"), []byte("x"), 0644)
	got := uniquePath(p)
	expected := filepath.Join(dir, "test_3.jpg")
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}
