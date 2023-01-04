package apikit

import (
	"errors"
	"net/http"
)

type BaseResponse struct {
	RequestID  string         `json:"request_id"`
	StatusCode int            `json:"status_code"`
	StatusText string         `json:"status_text"`
	Data       interface{}    `json:"data"`
	Error      string         `json:"error,omitempty"`
	Pagination *PaginationDTO `json:"pagination,omitempty"`
}

type PagedResponse struct {
	BaseResponse
	Pagination PaginationDTO `json:"pagination,omitempty"`
}

type PaginationDTO struct {
	Page  int `json:"page"`
	Total int `json:"total"`
}

var ResponseType = map[int]string{
	200: "success",
	400: "Bad Request",
	401: "Authentication Failure",
	403: "Forbidden",
	404: "Not Found",
	500: "Internal Server Error : Api Error",
	409: "Conflict Data",
}

// SuccessResponse output response 200
func SuccessResponse(requestID string, data interface{}, pagination ...PaginationDTO) BaseResponse {
	respon := BaseResponse{
		RequestID:  requestID,
		StatusCode: 200,
		StatusText: "success",
		Data:       data,
	}

	if len(pagination) > 0 {
		respon.Pagination = &pagination[0]
	}
	return respon
}

func ErrorResponse(requestID string, code int, err error) BaseResponse {
	if errors.Is(err, ErrBadRequest) {
		code = 400
	}

	if errors.Is(err, ErrInvalidUserPassword) {
		code = 401
	}

	if errors.Is(err, ErrKeynotFound) {
		code = http.StatusNotFound
	}

	return BaseResponse{
		RequestID:  requestID,
		StatusCode: code,
		StatusText: http.StatusText(code),
		Error:      err.Error(),
	}
}
