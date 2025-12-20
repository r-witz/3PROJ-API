package tmdb

type Configuration struct {
	Images     ImageConfiguration `json:"images"`
	ChangeKeys []string           `json:"change_keys"`
}

type ImageConfiguration struct {
	BaseURL       string   `json:"base_url"`
	SecureBaseURL string   `json:"secure_base_url"`
	BackdropSizes []string `json:"backdrop_sizes"`
	LogoSizes     []string `json:"logo_sizes"`
	PosterSizes   []string `json:"poster_sizes"`
	ProfileSizes  []string `json:"profile_sizes"`
	StillSizes    []string `json:"still_sizes"`
}

type Genre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type GenreListResponse struct {
	Genres []Genre `json:"genres"`
}

type MovieSummary struct {
	ID               int     `json:"id"`
	Title            string  `json:"title"`
	OriginalTitle    string  `json:"original_title"`
	OriginalLanguage string  `json:"original_language"`
	Overview         string  `json:"overview"`
	PosterPath       *string `json:"poster_path"`
	BackdropPath     *string `json:"backdrop_path"`
	ReleaseDate      string  `json:"release_date"`
	Adult            bool    `json:"adult"`
	GenreIDs         []int   `json:"genre_ids"`
	Popularity       float64 `json:"popularity"`
	VoteAverage      float64 `json:"vote_average"`
	VoteCount        int     `json:"vote_count"`
	Video            bool    `json:"video"`
}

type MovieDetails struct {
	ID                  int                 `json:"id"`
	IMDbID              *string             `json:"imdb_id"`
	Title               string              `json:"title"`
	OriginalTitle       string              `json:"original_title"`
	OriginalLanguage    string              `json:"original_language"`
	Overview            string              `json:"overview"`
	Tagline             string              `json:"tagline"`
	PosterPath          *string             `json:"poster_path"`
	BackdropPath        *string             `json:"backdrop_path"`
	ReleaseDate         string              `json:"release_date"`
	Runtime             *int                `json:"runtime"`
	Adult               bool                `json:"adult"`
	Genres              []Genre             `json:"genres"`
	Popularity          float64             `json:"popularity"`
	VoteAverage         float64             `json:"vote_average"`
	VoteCount           int                 `json:"vote_count"`
	Video               bool                `json:"video"`
	Budget              int64               `json:"budget"`
	Revenue             int64               `json:"revenue"`
	Homepage            *string             `json:"homepage"`
	Status              string              `json:"status"`
	ProductionCompanies []ProductionCompany `json:"production_companies"`
	ProductionCountries []ProductionCountry `json:"production_countries"`
	SpokenLanguages     []SpokenLanguage    `json:"spoken_languages"`
	BelongsToCollection *Collection         `json:"belongs_to_collection"`
}

type ProductionCompany struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	LogoPath      *string `json:"logo_path"`
	OriginCountry string  `json:"origin_country"`
}

type ProductionCountry struct {
	ISO3166_1 string `json:"iso_3166_1"`
	Name      string `json:"name"`
}

type SpokenLanguage struct {
	ISO639_1    string `json:"iso_639_1"`
	Name        string `json:"name"`
	EnglishName string `json:"english_name"`
}

type Collection struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	PosterPath   *string `json:"poster_path"`
	BackdropPath *string `json:"backdrop_path"`
}

type Credits struct {
	ID   int          `json:"id"`
	Cast []CastMember `json:"cast"`
	Crew []CrewMember `json:"crew"`
}

type CastMember struct {
	ID                 int     `json:"id"`
	Name               string  `json:"name"`
	OriginalName       string  `json:"original_name"`
	Character          string  `json:"character"`
	ProfilePath        *string `json:"profile_path"`
	CreditID           string  `json:"credit_id"`
	Order              int     `json:"order"`
	Adult              bool    `json:"adult"`
	Gender             int     `json:"gender"`
	KnownForDepartment string  `json:"known_for_department"`
	Popularity         float64 `json:"popularity"`
	CastID             int     `json:"cast_id"`
}

type CrewMember struct {
	ID                 int     `json:"id"`
	Name               string  `json:"name"`
	OriginalName       string  `json:"original_name"`
	Department         string  `json:"department"`
	Job                string  `json:"job"`
	ProfilePath        *string `json:"profile_path"`
	CreditID           string  `json:"credit_id"`
	Adult              bool    `json:"adult"`
	Gender             int     `json:"gender"`
	KnownForDepartment string  `json:"known_for_department"`
	Popularity         float64 `json:"popularity"`
}

type PaginatedResponse[T any] struct {
	Page         int `json:"page"`
	TotalPages   int `json:"total_pages"`
	TotalResults int `json:"total_results"`
	Results      []T `json:"results"`
}

type SearchMoviesResponse = PaginatedResponse[MovieSummary]
type DiscoverMoviesResponse = PaginatedResponse[MovieSummary]
type PopularMoviesResponse = PaginatedResponse[MovieSummary]
