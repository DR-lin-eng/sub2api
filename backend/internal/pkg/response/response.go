// Package response provides standardized HTTP response helpers.
package response

import (
	"log"
	"math"
	"net/http"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/util/logredact"
	"github.com/gin-gonic/gin"
)

// Response 标准API响应格式
type Response struct {
	Code     int               `json:"code"`
	Message  string            `json:"message"`
	Reason   string            `json:"reason,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Data     any               `json:"data,omitempty"`
}

// PaginatedData 分页数据格式（匹配前端期望）
type PaginatedData struct {
	Items    any   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Pages    int   `json:"pages"`
}

type jsonResponder interface {
	Request() *http.Request
	WriteJSON(status int, payload any)
}

type paginationQueryReader interface {
	QueryValue(name string) string
}

type ginResponder struct {
	ctx *gin.Context
}

func (g ginResponder) Request() *http.Request {
	if g.ctx == nil {
		return nil
	}
	return g.ctx.Request
}

func (g ginResponder) WriteJSON(status int, payload any) {
	if g.ctx == nil {
		return
	}
	g.ctx.JSON(status, payload)
}

type ginPaginationReader struct {
	ctx *gin.Context
}

func (g ginPaginationReader) QueryValue(name string) string {
	if g.ctx == nil {
		return ""
	}
	return g.ctx.Query(name)
}

// Success 返回成功响应
func Success(c *gin.Context, data any) {
	SuccessContext(ginResponder{ctx: c}, data)
}

// Created 返回创建成功响应
func Created(c *gin.Context, data any) {
	CreatedContext(ginResponder{ctx: c}, data)
}

// Accepted 返回异步接受响应 (HTTP 202)
func Accepted(c *gin.Context, data any) {
	AcceptedContext(ginResponder{ctx: c}, data)
}

// Error 返回错误响应
func Error(c *gin.Context, statusCode int, message string) {
	ErrorContext(ginResponder{ctx: c}, statusCode, message)
}

// ErrorWithDetails returns an error response compatible with the existing envelope while
// optionally providing structured error fields (reason/metadata).
func ErrorWithDetails(c *gin.Context, statusCode int, message, reason string, metadata map[string]string) {
	ErrorWithDetailsContext(ginResponder{ctx: c}, statusCode, message, reason, metadata)
}

// ErrorFrom converts an ApplicationError (or any error) into the envelope-compatible error response.
// It returns true if an error was written.
func ErrorFrom(c *gin.Context, err error) bool {
	return ErrorFromContext(ginResponder{ctx: c}, err)
}

// BadRequest 返回400错误
func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, message)
}

// Unauthorized 返回401错误
func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, message)
}

// Forbidden 返回403错误
func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, message)
}

// NotFound 返回404错误
func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, message)
}

// InternalError 返回500错误
func InternalError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, message)
}

// Paginated 返回分页数据
func Paginated(c *gin.Context, items any, total int64, page, pageSize int) {
	PaginatedContext(ginResponder{ctx: c}, items, total, page, pageSize)
}

// PaginationResult 分页结果（与pagination.PaginationResult兼容）
type PaginationResult struct {
	Total    int64
	Page     int
	PageSize int
	Pages    int
}

// PaginatedWithResult 使用PaginationResult返回分页数据
func PaginatedWithResult(c *gin.Context, items any, pagination *PaginationResult) {
	PaginatedWithResultContext(ginResponder{ctx: c}, items, pagination)
}

// ParsePagination 解析分页参数
func ParsePagination(c *gin.Context) (page, pageSize int) {
	return ParsePaginationValues(ginPaginationReader{ctx: c})
}

func SuccessContext(c jsonResponder, data any) {
	writeResponse(c, http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

func CreatedContext(c jsonResponder, data any) {
	writeResponse(c, http.StatusCreated, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

func AcceptedContext(c jsonResponder, data any) {
	writeResponse(c, http.StatusAccepted, Response{
		Code:    0,
		Message: "accepted",
		Data:    data,
	})
}

func ErrorContext(c jsonResponder, statusCode int, message string) {
	writeResponse(c, statusCode, Response{
		Code:     statusCode,
		Message:  message,
		Reason:   "",
		Metadata: nil,
	})
}

func ErrorWithDetailsContext(c jsonResponder, statusCode int, message, reason string, metadata map[string]string) {
	writeResponse(c, statusCode, Response{
		Code:     statusCode,
		Message:  message,
		Reason:   reason,
		Metadata: metadata,
	})
}

func ErrorFromContext(c jsonResponder, err error) bool {
	if err == nil {
		return false
	}

	statusCode, status := infraerrors.ToHTTP(err)

	if statusCode >= 500 {
		if req := c.Request(); req != nil {
			log.Printf("[ERROR] %s %s\n  Error: %s", req.Method, req.URL.Path, logredact.RedactText(err.Error()))
		}
	}

	ErrorWithDetailsContext(c, statusCode, status.Message, status.Reason, status.Metadata)
	return true
}

func PaginatedContext(c jsonResponder, items any, total int64, page, pageSize int) {
	pages := int(math.Ceil(float64(total) / float64(pageSize)))
	if pages < 1 {
		pages = 1
	}

	SuccessContext(c, PaginatedData{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Pages:    pages,
	})
}

func PaginatedWithResultContext(c jsonResponder, items any, pagination *PaginationResult) {
	if pagination == nil {
		SuccessContext(c, PaginatedData{
			Items:    items,
			Total:    0,
			Page:     1,
			PageSize: 20,
			Pages:    1,
		})
		return
	}

	SuccessContext(c, PaginatedData{
		Items:    items,
		Total:    pagination.Total,
		Page:     pagination.Page,
		PageSize: pagination.PageSize,
		Pages:    pagination.Pages,
	})
}

func ParsePaginationValues(c paginationQueryReader) (page, pageSize int) {
	page = 1
	pageSize = 20

	if p := c.QueryValue("page"); p != "" {
		if val, err := parseInt(p); err == nil && val > 0 {
			page = val
		}
	}

	// 支持 page_size 和 limit 两种参数名
	if ps := c.QueryValue("page_size"); ps != "" {
		if val, err := parseInt(ps); err == nil && val > 0 && val <= 1000 {
			pageSize = val
		}
	} else if l := c.QueryValue("limit"); l != "" {
		if val, err := parseInt(l); err == nil && val > 0 && val <= 1000 {
			pageSize = val
		}
	}

	return page, pageSize
}

func writeResponse(c jsonResponder, status int, payload Response) {
	if c == nil {
		return
	}
	c.WriteJSON(status, payload)
}

func parseInt(s string) (int, error) {
	var result int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil
		}
		result = result*10 + int(c-'0')
	}
	return result, nil
}
