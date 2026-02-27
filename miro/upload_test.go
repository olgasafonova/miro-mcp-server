package miro

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// =============================================================================
// ValidateUploadPath Tests
// =============================================================================

func TestValidateUploadPath_ValidFileInCWD(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.png")
	if err := os.WriteFile(tmpFile, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}

	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	resolved, err := ValidateUploadPath(tmpFile)
	if err != nil {
		t.Fatalf("ValidateUploadPath() error = %v", err)
	}
	if resolved == "" {
		t.Error("resolved path should not be empty")
	}
}

func TestValidateUploadPath_PathTraversal(t *testing.T) {
	tmpDir := t.TempDir()

	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	t.Setenv("MIRO_UPLOAD_ALLOWED_DIRS", "")

	tests := []struct {
		name string
		path string
	}{
		{"parent traversal", "../../../etc/passwd"},
		{"absolute etc", "/etc/passwd"},
		{"dot-dot in middle", filepath.Join(tmpDir, "..", "etc", "passwd")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateUploadPath(tt.path)
			if err == nil {
				t.Error("expected error for path traversal attempt")
			}
			// Error can be either "outside allowed directories" (path exists but outside)
			// or "failed to resolve symlinks" (path doesn't exist). Both block the attack.
		})
	}
}

func TestValidateUploadPath_Symlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks require admin on Windows")
	}

	tmpDir := t.TempDir()
	outsideDir := t.TempDir()

	outsideFile := filepath.Join(outsideDir, "secret.txt")
	if err := os.WriteFile(outsideFile, []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}

	symlinkPath := filepath.Join(tmpDir, "link.txt")
	if err := os.Symlink(outsideFile, symlinkPath); err != nil {
		t.Fatal(err)
	}

	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	t.Setenv("MIRO_UPLOAD_ALLOWED_DIRS", "")

	_, err := ValidateUploadPath(symlinkPath)
	if err == nil {
		t.Error("expected error for symlink pointing outside allowed directories")
	}
}

func TestValidateUploadPath_AllowedDirsEnv(t *testing.T) {
	allowedDir := t.TempDir()
	testFile := filepath.Join(allowedDir, "doc.pdf")
	if err := os.WriteFile(testFile, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}

	otherDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(otherDir)

	t.Setenv("MIRO_UPLOAD_ALLOWED_DIRS", allowedDir)

	resolved, err := ValidateUploadPath(testFile)
	if err != nil {
		t.Fatalf("ValidateUploadPath() error = %v", err)
	}
	if resolved == "" {
		t.Error("resolved path should not be empty")
	}
}

func TestValidateUploadPath_MultipleAllowedDirs(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	testFile := filepath.Join(dir2, "image.png")
	if err := os.WriteFile(testFile, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}

	otherDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(otherDir)

	t.Setenv("MIRO_UPLOAD_ALLOWED_DIRS", dir1+","+dir2)

	resolved, err := ValidateUploadPath(testFile)
	if err != nil {
		t.Fatalf("ValidateUploadPath() error = %v", err)
	}
	if resolved == "" {
		t.Error("resolved path should not be empty")
	}
}

func TestValidateUploadPath_NonexistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	_, err := ValidateUploadPath(filepath.Join(tmpDir, "nonexistent.png"))
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}
