package utils

import "github.com/gin-gonic/gin"

type ApiResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type PageData struct {
	List       interface{} `json:"list"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"pageSize"`
	TotalPages int         `json:"totalPages"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(200, ApiResponse{Code: 0, Message: "ok", Data: data})
}

func SuccessMsg(c *gin.Context, data interface{}, msg string) {
	c.JSON(200, ApiResponse{Code: 0, Message: msg, Data: data})
}

func Fail(c *gin.Context, code int, msg string) {
	c.JSON(200, ApiResponse{Code: code, Message: msg, Data: nil})
}

func FailStatus(c *gin.Context, httpStatus int, code int, msg string) {
	c.JSON(httpStatus, ApiResponse{Code: code, Message: msg, Data: nil})
}

func Paginated(c *gin.Context, list interface{}, total, page, pageSize int) {
	totalPages := (total + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	c.JSON(200, ApiResponse{
		Code:    0,
		Message: "ok",
		Data: PageData{
			List:       list,
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: totalPages,
		},
	})
}

func GetAuthUserID(c *gin.Context) int {
	if id, ok := c.Get("userId"); ok {
		if v, ok := id.(int); ok {
			return v
		}
	}
	return 0
}

func GetAuthUserRole(c *gin.Context) string {
	if role, ok := c.Get("userRole"); ok {
		if v, ok := role.(string); ok {
			return v
		}
	}
	return ""
}
