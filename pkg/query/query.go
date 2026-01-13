package query

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type Params struct {
	Page      int
	PerPage   int
	Sort      string
	SortField string
	SortOrder string
}

type Config struct {
	DefaultPage    int
	DefaultPerPage int
	MaxPerPage     int
	AllowedSorts   []string
}

func Parse(c *gin.Context, config Config) (*Params, error) {
	params := &Params{
		Page:    config.DefaultPage,
		PerPage: config.DefaultPerPage,
	}

	if config.DefaultPage == 0 {
		params.Page = 1
	}
	if config.DefaultPerPage == 0 {
		params.PerPage = 20
	}
	if config.MaxPerPage == 0 {
		config.MaxPerPage = 100
	}

	if pageStr := c.Query("page"); pageStr != "" {
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			return nil, fmt.Errorf("invalid page parameter")
		}
		params.Page = page
	}

	if perPageStr := c.Query("per_page"); perPageStr != "" {
		perPage, err := strconv.Atoi(perPageStr)
		if err != nil || perPage < 1 {
			return nil, fmt.Errorf("invalid per_page parameter")
		}
		if perPage > config.MaxPerPage {
			perPage = config.MaxPerPage
		}
		params.PerPage = perPage
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

func (p *Params) Offset() int {
	return (p.Page - 1) * p.PerPage
}
