package tmdb

import "fmt"

const (
	BaseURL = "https://api.themoviedb.org/3"

	DefaultImageBaseURL = "https://image.tmdb.org/t/p/"

	PosterSizeSmall    = "w185"
	PosterSizeMedium   = "w342"
	PosterSizeLarge    = "w500"
	PosterSizeOriginal = "original"

	BackdropSizeSmall    = "w300"
	BackdropSizeMedium   = "w780"
	BackdropSizeLarge    = "w1280"
	BackdropSizeOriginal = "original"

	ProfileSizeSmall    = "w45"
	ProfileSizeMedium   = "w185"
	ProfileSizeLarge    = "h632"
	ProfileSizeOriginal = "original"
)

type ImageURLBuilder struct {
	baseURL string
}

func NewImageURLBuilder(baseURL string) *ImageURLBuilder {
	if baseURL == "" {
		baseURL = DefaultImageBaseURL
	}
	return &ImageURLBuilder{baseURL: baseURL}
}

func (b *ImageURLBuilder) PosterURL(path *string, size string) string {
	if path == nil || *path == "" {
		return ""
	}
	return fmt.Sprintf("%s%s%s", b.baseURL, size, *path)
}

func (b *ImageURLBuilder) BackdropURL(path *string, size string) string {
	if path == nil || *path == "" {
		return ""
	}
	return fmt.Sprintf("%s%s%s", b.baseURL, size, *path)
}

func (b *ImageURLBuilder) ProfileURL(path *string, size string) string {
	if path == nil || *path == "" {
		return ""
	}
	return fmt.Sprintf("%s%s%s", b.baseURL, size, *path)
}

type SortBy string

const (
	SortByPopularityAsc          SortBy = "popularity.asc"
	SortByPopularityDesc         SortBy = "popularity.desc"
	SortByReleaseDateAsc         SortBy = "release_date.asc"
	SortByReleaseDateDesc        SortBy = "release_date.desc"
	SortByRevenueAsc             SortBy = "revenue.asc"
	SortByRevenueDesc            SortBy = "revenue.desc"
	SortByPrimaryReleaseDateAsc  SortBy = "primary_release_date.asc"
	SortByPrimaryReleaseDateDesc SortBy = "primary_release_date.desc"
	SortByOriginalTitleAsc       SortBy = "original_title.asc"
	SortByOriginalTitleDesc      SortBy = "original_title.desc"
	SortByVoteAverageAsc         SortBy = "vote_average.asc"
	SortByVoteAverageDesc        SortBy = "vote_average.desc"
	SortByVoteCountAsc           SortBy = "vote_count.asc"
	SortByVoteCountDesc          SortBy = "vote_count.desc"
)
