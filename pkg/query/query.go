package query

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	DefaultLimit = 20
	MaxLimit     = 100
)

type Params struct {
	Offset    int
	Limit     int
	Sort      string
	SortField string
	SortOrder string
}

type Config struct {
	DefaultLimit int
	MaxLimit     int
	AllowedSorts []string
}

func Parse(c *gin.Context, config Config) (*Params, error) {
	params := &Params{
		Offset: 0,
		Limit:  config.DefaultLimit,
	}

	if config.DefaultLimit == 0 {
		params.Limit = DefaultLimit
	}
	if config.MaxLimit == 0 {
		config.MaxLimit = MaxLimit
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			return nil, fmt.Errorf("invalid offset parameter")
		}
		params.Offset = offset
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 {
			return nil, fmt.Errorf("invalid limit parameter")
		}
		if limit > config.MaxLimit {
			limit = config.MaxLimit
		}
		params.Limit = limit
	}

	if sortStr := c.Query("sort"); sortStr != "" {
		params.Sort = sortStr

		if strings.HasPrefix(sortStr, "-") {
			params.SortField = sortStr[1:]
			params.SortOrder = "DESC"
		} else if strings.HasPrefix(sortStr, "+") {
			params.SortField = sortStr[1:]
			params.SortOrder = "ASC"
		} else {
			params.SortField = sortStr
			params.SortOrder = "ASC"
		}

		if len(config.AllowedSorts) > 0 && !slices.Contains(config.AllowedSorts, params.SortField) {
			return nil, fmt.Errorf("invalid sort field: %s", params.SortField)
		}
	}

	return params, nil
}
