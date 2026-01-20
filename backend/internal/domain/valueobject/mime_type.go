package valueobject

import (
	"errors"
	"strings"
)

var (
	ErrInvalidMimeType = errors.New("invalid MIME type")
)

// MimeCategory はMIMEタイプのカテゴリを表す
type MimeCategory string

const (
	MimeCategoryImage    MimeCategory = "image"
	MimeCategoryVideo    MimeCategory = "video"
	MimeCategoryAudio    MimeCategory = "audio"
	MimeCategoryDocument MimeCategory = "document"
	MimeCategoryArchive  MimeCategory = "archive"
	MimeCategoryOther    MimeCategory = "other"
)

// MimeType はMIMEタイプを表す値オブジェクト
type MimeType struct {
	value    string
	category MimeCategory
}

// NewMimeType は文字列からMimeTypeを生成します
func NewMimeType(mimeType string) (MimeType, error) {
	trimmed := strings.TrimSpace(mimeType)

	// 空チェック
	if trimmed == "" {
		return MimeType{}, ErrInvalidMimeType
	}

	// 基本的な形式チェック（type/subtype）
	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return MimeType{}, ErrInvalidMimeType
	}

	value := strings.ToLower(trimmed)
	category := categorizeFromMimeType(value)

	return MimeType{value: value, category: category}, nil
}

// ReconstructMimeType はDBからMimeTypeを復元します
func ReconstructMimeType(value string, category MimeCategory) MimeType {
	return MimeType{value: value, category: category}
}

// categorizeFromMimeType はMIMEタイプからカテゴリを判定します
func categorizeFromMimeType(mimeType string) MimeCategory {
	parts := strings.Split(mimeType, "/")
	if len(parts) == 0 {
		return MimeCategoryOther
	}

	mainType := parts[0]
	subType := ""
	if len(parts) > 1 {
		subType = parts[1]
	}

	switch mainType {
	case "image":
		return MimeCategoryImage
	case "video":
		return MimeCategoryVideo
	case "audio":
		return MimeCategoryAudio
	case "text":
		return MimeCategoryDocument
	case "application":
		// ドキュメント系
		if strings.Contains(subType, "pdf") ||
			strings.Contains(subType, "msword") ||
			strings.Contains(subType, "document") ||
			strings.Contains(subType, "spreadsheet") ||
			strings.Contains(subType, "presentation") ||
			strings.Contains(subType, "json") ||
			strings.Contains(subType, "xml") {
			return MimeCategoryDocument
		}
		// アーカイブ系
		if strings.Contains(subType, "zip") ||
			strings.Contains(subType, "tar") ||
			strings.Contains(subType, "gzip") ||
			strings.Contains(subType, "rar") ||
			strings.Contains(subType, "7z") ||
			strings.Contains(subType, "compress") {
			return MimeCategoryArchive
		}
		return MimeCategoryOther
	default:
		return MimeCategoryOther
	}
}

// Value は値を返します
func (m MimeType) Value() string {
	return m.value
}

// Category はカテゴリを返します
func (m MimeType) Category() MimeCategory {
	return m.category
}

// String は文字列を返します（Stringerインターフェース）
func (m MimeType) String() string {
	return m.value
}

// Type はMIMEタイプの主タイプを返します（例: "text", "image"）
func (m MimeType) Type() string {
	parts := strings.Split(m.value, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// Subtype はMIMEタイプのサブタイプを返します（例: "plain", "png"）
func (m MimeType) Subtype() string {
	parts := strings.Split(m.value, "/")
	if len(parts) > 1 {
		// パラメータを除去（例: text/plain; charset=utf-8 → plain）
		subtype := parts[1]
		if idx := strings.Index(subtype, ";"); idx != -1 {
			subtype = subtype[:idx]
		}
		return strings.TrimSpace(subtype)
	}
	return ""
}

// IsImage は画像MIMEタイプかどうかを判定します
func (m MimeType) IsImage() bool {
	return m.category == MimeCategoryImage
}

// IsVideo は動画MIMEタイプかどうかを判定します
func (m MimeType) IsVideo() bool {
	return m.category == MimeCategoryVideo
}

// IsAudio は音声MIMEタイプかどうかを判定します
func (m MimeType) IsAudio() bool {
	return m.category == MimeCategoryAudio
}

// IsDocument はドキュメントMIMEタイプかどうかを判定します
func (m MimeType) IsDocument() bool {
	return m.category == MimeCategoryDocument
}

// IsArchive はアーカイブMIMEタイプかどうかを判定します
func (m MimeType) IsArchive() bool {
	return m.category == MimeCategoryArchive
}

// IsPDF はPDFかどうかを判定します
func (m MimeType) IsPDF() bool {
	return m.value == "application/pdf"
}

// IsPreviewable はプレビュー可能かどうかを判定します
func (m MimeType) IsPreviewable() bool {
	return m.IsImage() || m.IsPDF() || m.Type() == "text" || m.value == "application/json"
}

// Equals は等価性を判定します
func (m MimeType) Equals(other MimeType) bool {
	return m.value == other.value
}

// 一般的なMIMEタイプ定数
var (
	MimeTypeOctetStream = MimeType{value: "application/octet-stream", category: MimeCategoryOther}
	MimeTypePDF         = MimeType{value: "application/pdf", category: MimeCategoryDocument}
	MimeTypeJSON        = MimeType{value: "application/json", category: MimeCategoryDocument}
	MimeTypeZip         = MimeType{value: "application/zip", category: MimeCategoryArchive}
	MimeTypeTextPlain   = MimeType{value: "text/plain", category: MimeCategoryDocument}
	MimeTypeTextHTML    = MimeType{value: "text/html", category: MimeCategoryDocument}
	MimeTypeImagePNG    = MimeType{value: "image/png", category: MimeCategoryImage}
	MimeTypeImageJPEG   = MimeType{value: "image/jpeg", category: MimeCategoryImage}
	MimeTypeImageGIF    = MimeType{value: "image/gif", category: MimeCategoryImage}
	MimeTypeImageWebP   = MimeType{value: "image/webp", category: MimeCategoryImage}
	MimeTypeVideoMP4    = MimeType{value: "video/mp4", category: MimeCategoryVideo}
	MimeTypeAudioMP3    = MimeType{value: "audio/mpeg", category: MimeCategoryAudio}
)
