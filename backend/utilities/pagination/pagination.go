package pagination

import (
	"net/http"
	"strconv"

	"passenger-go/backend/schemas"
)

type Pagination struct {
	Page int `json:"page"`
	Take int `json:"take"`
}

func PaginationParams(query *http.Request) (*Pagination, error) {
	page := query.URL.Query().Get("page")
	if page == "" {
		page = "1"
	}

	pageInt, err := strconv.Atoi(page)
	if err != nil {
		return nil, schemas.NewAPIError(schemas.ErrInvalidRequest, "Invalid page", err)
	}

	take := query.URL.Query().Get("take")
	if take == "" {
		take = "10"
	}

	takeInt, err := strconv.Atoi(take)
	if err != nil {
		return nil, schemas.NewAPIError(schemas.ErrInvalidRequest, "Invalid take", err)
	}

	return &Pagination{
		Page: pageInt,
		Take: takeInt,
	}, nil
}
