package presenter

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Response は統一レスポンス構造を定義します
type Response struct {
	Data interface{} `json:"data"`
	Meta interface{} `json:"meta"`
}

// Pagination はページネーション情報を定義します
type Pagination struct {
	Page       int  `json:"page"`
	PerPage    int  `json:"per_page"`
	TotalItems int  `json:"total_items"`
	TotalPages int  `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}

// Meta はメタ情報を定義します
type Meta struct {
	Message    string      `json:"message,omitempty"`
	Pagination *Pagination `json:"pagination,omitempty"`
}

// OK は成功レスポンスを返します
func OK(c echo.Context, data interface{}) error {
	return c.JSON(http.StatusOK, Response{
		Data: data,
		Meta: nil,
	})
}

// OKWithMeta はメタ情報付き成功レスポンスを返します
func OKWithMeta(c echo.Context, data interface{}, meta interface{}) error {
	return c.JSON(http.StatusOK, Response{
		Data: data,
		Meta: meta,
	})
}

// Created は作成成功レスポンスを返します
func Created(c echo.Context, data interface{}) error {
	return c.JSON(http.StatusCreated, Response{
		Data: data,
		Meta: nil,
	})
}

// NoContent はコンテンツなしレスポンスを返します
func NoContent(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

// Deleted は削除成功レスポンスを返します
func Deleted(c echo.Context, message string) error {
	return c.JSON(http.StatusOK, Response{
		Data: nil,
		Meta: Meta{Message: message},
	})
}

// List はリスト取得レスポンスを返します
func List(c echo.Context, data interface{}, pagination *Pagination) error {
	return c.JSON(http.StatusOK, Response{
		Data: data,
		Meta: Meta{Pagination: pagination},
	})
}

// NewPagination はページネーション情報を作成します
func NewPagination(page, perPage, totalItems int) *Pagination {
	totalPages := (totalItems + perPage - 1) / perPage
	if totalPages == 0 {
		totalPages = 1
	}

	return &Pagination{
		Page:       page,
		PerPage:    perPage,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}

// DefaultPage はデフォルトのページ番号を返します
const DefaultPage = 1

// DefaultPerPage はデフォルトの1ページあたりの件数を返します
const DefaultPerPage = 20

// MaxPerPage は最大の1ページあたりの件数を返します
const MaxPerPage = 100

// NormalizePagination はページネーションパラメータを正規化します
func NormalizePagination(page, perPage int) (int, int) {
	if page < 1 {
		page = DefaultPage
	}
	if perPage < 1 {
		perPage = DefaultPerPage
	}
	if perPage > MaxPerPage {
		perPage = MaxPerPage
	}
	return page, perPage
}

// Offset はオフセット値を計算します
func Offset(page, perPage int) int {
	return (page - 1) * perPage
}
