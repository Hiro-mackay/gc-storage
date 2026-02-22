package valueobject

import (
	"strings"
	"testing"
)

func TestNewFolderName_ValidName_ReturnsFolderName(t *testing.T) {
	fn, err := NewFolderName("Documents")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fn.Value() != "Documents" {
		t.Errorf("got %q, want %q", fn.Value(), "Documents")
	}
}

func TestNewFolderName_EmptyString_ReturnsErrFolderNameEmpty(t *testing.T) {
	_, err := NewFolderName("")

	if err != ErrFolderNameEmpty {
		t.Errorf("expected ErrFolderNameEmpty, got: %v", err)
	}
}

func TestNewFolderName_WhitespaceOnly_ReturnsErrFolderNameEmpty(t *testing.T) {
	_, err := NewFolderName("   ")

	if err != ErrFolderNameEmpty {
		t.Errorf("expected ErrFolderNameEmpty, got: %v", err)
	}
}

func TestNewFolderName_LeadingTrailingSpaces_TrimsAndSucceeds(t *testing.T) {
	fn, err := NewFolderName("  hello  ")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fn.Value() != "hello" {
		t.Errorf("got %q, want %q", fn.Value(), "hello")
	}
}

func TestNewFolderName_DotReserved_ReturnsErrFolderNameReserved(t *testing.T) {
	_, err := NewFolderName(".")

	if err != ErrFolderNameReserved {
		t.Errorf("expected ErrFolderNameReserved, got: %v", err)
	}
}

func TestNewFolderName_DotDotReserved_ReturnsErrFolderNameReserved(t *testing.T) {
	_, err := NewFolderName("..")

	if err != ErrFolderNameReserved {
		t.Errorf("expected ErrFolderNameReserved, got: %v", err)
	}
}

func TestNewFolderName_ExactlyMaxLength_Succeeds(t *testing.T) {
	name := strings.Repeat("a", FolderNameMaxBytes)

	fn, err := NewFolderName(name)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fn.Value() != name {
		t.Errorf("value mismatch for max-length name")
	}
}

func TestNewFolderName_ExceedsMaxLength_ReturnsErrFolderNameTooLong(t *testing.T) {
	name := strings.Repeat("a", FolderNameMaxBytes+1)

	_, err := NewFolderName(name)

	if err != ErrFolderNameTooLong {
		t.Errorf("expected ErrFolderNameTooLong, got: %v", err)
	}
}

func TestNewFolderName_ContainsSlash_ReturnsErrFolderNameForbiddenChars(t *testing.T) {
	_, err := NewFolderName("my/folder")

	if err != ErrFolderNameForbiddenChars {
		t.Errorf("expected ErrFolderNameForbiddenChars, got: %v", err)
	}
}

func TestNewFolderName_ContainsBackslash_ReturnsErrFolderNameForbiddenChars(t *testing.T) {
	_, err := NewFolderName("my\\folder")

	if err != ErrFolderNameForbiddenChars {
		t.Errorf("expected ErrFolderNameForbiddenChars, got: %v", err)
	}
}

func TestNewFolderName_ContainsColon_ReturnsErrFolderNameForbiddenChars(t *testing.T) {
	_, err := NewFolderName("my:folder")

	if err != ErrFolderNameForbiddenChars {
		t.Errorf("expected ErrFolderNameForbiddenChars, got: %v", err)
	}
}

func TestNewFolderName_ContainsAsterisk_ReturnsErrFolderNameForbiddenChars(t *testing.T) {
	_, err := NewFolderName("my*folder")

	if err != ErrFolderNameForbiddenChars {
		t.Errorf("expected ErrFolderNameForbiddenChars, got: %v", err)
	}
}

func TestNewFolderName_ContainsQuestionMark_ReturnsErrFolderNameForbiddenChars(t *testing.T) {
	_, err := NewFolderName("my?folder")

	if err != ErrFolderNameForbiddenChars {
		t.Errorf("expected ErrFolderNameForbiddenChars, got: %v", err)
	}
}

func TestNewFolderName_ContainsDoubleQuote_ReturnsErrFolderNameForbiddenChars(t *testing.T) {
	_, err := NewFolderName(`my"folder`)

	if err != ErrFolderNameForbiddenChars {
		t.Errorf("expected ErrFolderNameForbiddenChars, got: %v", err)
	}
}

func TestNewFolderName_ContainsLessThan_ReturnsErrFolderNameForbiddenChars(t *testing.T) {
	_, err := NewFolderName("my<folder")

	if err != ErrFolderNameForbiddenChars {
		t.Errorf("expected ErrFolderNameForbiddenChars, got: %v", err)
	}
}

func TestNewFolderName_ContainsGreaterThan_ReturnsErrFolderNameForbiddenChars(t *testing.T) {
	_, err := NewFolderName("my>folder")

	if err != ErrFolderNameForbiddenChars {
		t.Errorf("expected ErrFolderNameForbiddenChars, got: %v", err)
	}
}

func TestNewFolderName_ContainsPipe_ReturnsErrFolderNameForbiddenChars(t *testing.T) {
	_, err := NewFolderName("my|folder")

	if err != ErrFolderNameForbiddenChars {
		t.Errorf("expected ErrFolderNameForbiddenChars, got: %v", err)
	}
}

func TestFolderName_Value_ReturnsUnderlyingString(t *testing.T) {
	fn, _ := NewFolderName("Photos")

	if fn.Value() != "Photos" {
		t.Errorf("got %q, want %q", fn.Value(), "Photos")
	}
}

func TestFolderName_String_ReturnsUnderlyingString(t *testing.T) {
	fn, _ := NewFolderName("Photos")

	if fn.String() != "Photos" {
		t.Errorf("got %q, want %q", fn.String(), "Photos")
	}
}

func TestFolderName_IsEmpty_NonEmptyValue_ReturnsFalse(t *testing.T) {
	fn, _ := NewFolderName("Photos")

	if fn.IsEmpty() {
		t.Error("IsEmpty should return false for non-empty FolderName")
	}
}

func TestFolderName_IsEmpty_ZeroValue_ReturnsTrue(t *testing.T) {
	var fn FolderName

	if !fn.IsEmpty() {
		t.Error("IsEmpty should return true for zero-value FolderName")
	}
}

func TestFolderName_Equals_SameName_ReturnsTrue(t *testing.T) {
	fn1, _ := NewFolderName("Documents")
	fn2, _ := NewFolderName("Documents")

	if !fn1.Equals(fn2) {
		t.Error("Equals should return true for same name")
	}
}

func TestFolderName_Equals_DifferentName_ReturnsFalse(t *testing.T) {
	fn1, _ := NewFolderName("Documents")
	fn2, _ := NewFolderName("Photos")

	if fn1.Equals(fn2) {
		t.Error("Equals should return false for different names")
	}
}

func TestNewFolderName_UnicodeCharacters_Succeeds(t *testing.T) {
	fn, err := NewFolderName("ドキュメント")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fn.Value() != "ドキュメント" {
		t.Errorf("got %q, want %q", fn.Value(), "ドキュメント")
	}
}
