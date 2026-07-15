package kit

import (
	"os"
	"testing"
)

func TestValidateSavedPathPartAllowsExistingPresetNameStyle(t *testing.T) {
	valid := []string{
		"!!Aqua Space_Time Tunnel",
		"(!) Cedar Space_Square Trails",
		"{(!) Ready_Player Four",
		"_Boot",
		"_Current",
		"archive",
	}
	for _, value := range valid {
		t.Run(value, func(t *testing.T) {
			if err := validateSavedPathPart("filename", value); err != nil {
				t.Fatalf("validateSavedPathPart rejected %q: %v", value, err)
			}
		})
	}
}

func TestValidateSavedPathPartRejectsUnsafeValues(t *testing.T) {
	invalid := []string{
		"",
		".",
		"..",
		"../escape",
		`..\escape`,
		`folder\file`,
		"folder/file",
		"name:stream",
		"bad*name",
		"bad?name",
		`bad"name`,
		"bad<name",
		"bad>name",
		"bad|name",
		"trailing.",
		"trailing ",
		" leading",
		"CON",
		"nul.txt",
		"COM1",
		"LPT9",
		"bad\x00name",
	}
	for _, value := range invalid {
		t.Run(value, func(t *testing.T) {
			if err := validateSavedPathPart("filename", value); err == nil {
				t.Fatalf("validateSavedPathPart accepted unsafe value %q", value)
			}
		})
	}
}

func TestReadableSavedFilePathRejectsUnsafeCategoryAndSuffix(t *testing.T) {
	if _, err := ReadableSavedFilePath("../global", "_Boot", ".json"); err == nil {
		t.Fatalf("ReadableSavedFilePath accepted unsafe category")
	}
	if _, err := ReadableSavedFilePath("global", "_Boot", "../bad"); err == nil {
		t.Fatalf("ReadableSavedFilePath accepted unsafe suffix")
	}
}

func writeTestPreset(t *testing.T, category string, filename string) {
	t.Helper()
	path, err := WritableSavedFilePath(category, filename, ".json")
	if err != nil {
		t.Fatalf("WritableSavedFilePath(%q,%q): %v", category, filename, err)
	}
	if err := os.WriteFile(path, []byte("{\"params\":{}}\n"), 0644); err != nil {
		t.Fatalf("write preset: %v", err)
	}
}

func savedPresetExists(t *testing.T, category string, filename string) bool {
	t.Helper()
	path, err := ReadableSavedFilePath(category, filename, ".json")
	if err != nil {
		t.Fatalf("ReadableSavedFilePath(%q,%q): %v", category, filename, err)
	}
	return PathExists(path)
}

func TestRenameSavedFile(t *testing.T) {
	t.Setenv("PALETTE_DATAROOT", t.TempDir())
	InitLog("")

	writeTestPreset(t, "quad", "Foo")

	if err := RenameSavedFile("quad", "Foo", "Bar"); err != nil {
		t.Fatalf("RenameSavedFile: %v", err)
	}
	if savedPresetExists(t, "quad", "Foo") {
		t.Fatal("original preset should be gone after rename")
	}
	if !savedPresetExists(t, "quad", "Bar") {
		t.Fatal("renamed preset should exist")
	}

	// Renaming onto an existing preset must fail without clobbering it.
	writeTestPreset(t, "quad", "Keep")
	if err := RenameSavedFile("quad", "Bar", "Keep"); err == nil {
		t.Fatal("rename onto existing name should fail")
	}
	if !savedPresetExists(t, "quad", "Bar") {
		t.Fatal("source preset should survive a failed rename")
	}
}

func TestCopySavedFile(t *testing.T) {
	t.Setenv("PALETTE_DATAROOT", t.TempDir())
	InitLog("")

	writeTestPreset(t, "quad", "Groove")

	if err := CopySavedFile("quad", "Groove", "quad_chill"); err != nil {
		t.Fatalf("CopySavedFile: %v", err)
	}
	if !savedPresetExists(t, "quad", "Groove") {
		t.Fatal("original preset should remain after copy")
	}
	if !savedPresetExists(t, "quad_chill", "Groove") {
		t.Fatal("copied preset should exist in the destination theme")
	}

	// Copying onto an existing destination must fail without clobbering it.
	if err := CopySavedFile("quad", "Groove", "quad_chill"); err == nil {
		t.Fatal("copy onto existing name should fail")
	}
}

func TestMoveSavedFile(t *testing.T) {
	t.Setenv("PALETTE_DATAROOT", t.TempDir())
	InitLog("")

	writeTestPreset(t, "quad", "Groove")

	if err := MoveSavedFile("quad", "Groove", "quad_chill"); err != nil {
		t.Fatalf("MoveSavedFile: %v", err)
	}
	if savedPresetExists(t, "quad", "Groove") {
		t.Fatal("preset should no longer be in the source theme")
	}
	if !savedPresetExists(t, "quad_chill", "Groove") {
		t.Fatal("preset should exist in the destination theme")
	}

	// Moving onto an existing destination must fail without clobbering it.
	writeTestPreset(t, "quad", "Groove")
	if err := MoveSavedFile("quad", "Groove", "quad_chill"); err == nil {
		t.Fatal("move onto existing name should fail")
	}
	if !savedPresetExists(t, "quad", "Groove") {
		t.Fatal("source preset should survive a failed move")
	}
}
