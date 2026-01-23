package tmdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

type SearchPersonParams struct {
	Query        string
	Page         int
	Language     string
	IncludeAdult bool
}

func (c *Client) SearchPerson(ctx context.Context, params SearchPersonParams) (*SearchPersonResponse, error) {
	if params.Query == "" {
		return nil, ErrInvalidRequest
	}

	queryParams := url.Values{}
	queryParams.Set("query", params.Query)

	if params.Page > 0 {
		queryParams.Set("page", strconv.Itoa(params.Page))
	}
	if params.Language != "" {
		queryParams.Set("language", params.Language)
	}
	if params.IncludeAdult {
		queryParams.Set("include_adult", "true")
	}

	body, err := c.doRequest(ctx, "GET", "/search/person", queryParams)
	if err != nil {
		return nil, err
	}

	var resp SearchPersonResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, &RequestError{Operation: "/search/person", Err: err}
	}

	return &resp, nil
}

func (c *Client) GetPersonDetails(ctx context.Context, personID int, language string) (*PersonDetails, error) {
	params := url.Values{}
	if language != "" {
		params.Set("language", language)
	}

	endpoint := fmt.Sprintf("/person/%d", personID)
	body, err := c.doRequest(ctx, "GET", endpoint, params)
	if err != nil {
		return nil, err
	}

	var person PersonDetails
	if err := json.Unmarshal(body, &person); err != nil {
		return nil, &RequestError{Operation: endpoint, Err: err}
	}

	return &person, nil
}

func (c *Client) GetPersonMovieCredits(ctx context.Context, personID int, language string) (*PersonMovieCredits, error) {
	params := url.Values{}
	if language != "" {
		params.Set("language", language)
	}

	endpoint := fmt.Sprintf("/person/%d/movie_credits", personID)
	body, err := c.doRequest(ctx, "GET", endpoint, params)
	if err != nil {
		return nil, err
	}

	var credits PersonMovieCredits
	if err := json.Unmarshal(body, &credits); err != nil {
		return nil, &RequestError{Operation: endpoint, Err: err}
	}

	return &credits, nil
}
