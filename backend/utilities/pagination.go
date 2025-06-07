package utilities

import (
	"net/http"
	"strconv"

	"passenger-go/backend/schemas"
)

type Pagination struct {
	Page int `json:"page"`
	Take int `json:"take"`
}

func PaginationParams(query *http.Request) (*Pagination, *schemas.APIError) {
	page, err := strconv.Atoi(query.URL.Query().Get("page"))
	if err != nil {
		return nil, schemas.NewAPIError(schemas.ErrInvalidRequest, "Invalid page", err)
	}

	take, err := strconv.Atoi(query.URL.Query().Get("take"))
	if err != nil {
		return nil, schemas.NewAPIError(schemas.ErrInvalidRequest, "Invalid take", err)
	}

	return &Pagination{
		Page: page,
		Take: take,
	}, nil
}
