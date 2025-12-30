package api

type BaseResponse[T any] struct {
	RequestID  string         `json:"request_id"`
	Message    string         `json:"message,omitempty"`
	Error      *string        `json:"error,omitempty"`
	Data       T              `json:"data"`
	Pagination *PaginationDTO `json:"pagination,omitempty"`
}

type ListItem struct {
	Value string `json:"value"`
	Label string `json:"label"`
	Count *int   `json:"count,omitempty"`
}

type PaginationDTO struct {
	Page  uint `json:"page"`
	Total uint `json:"total"`
}

type PagedData[T any] struct {
	Data       T
	Pagination PaginationDTO
}

type ByIDRequestDTO[T comparable] struct {
	ID T `query:"id" json:"id"`
}

func SuccessResponse[T any](requestID string, data T, pagination ...PaginationDTO) BaseResponse[T] {
	respon := BaseResponse[T]{
		RequestID: requestID,
		Message:   "success",
		Data:      data,
	}

	if len(pagination) > 0 {
		respon.Pagination = &pagination[0]
	}
	return respon
}

func ErrorResponse(requestID string, err error) BaseResponse[any] {
	error := err.Error()
	return BaseResponse[any]{
		RequestID: requestID,
		Error:     &error,
	}
}
