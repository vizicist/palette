package kit

import "testing"

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
