package services

import (
	"strconv"
	"strings"
	"unicode"

	"duskforge-api/pkg/tmdb"
)

const matchThreshold = 80

// bestMatch finds the highest-scoring movie from TMDB search results.
// Returns the best match and its score. Score < matchThreshold means no good match.
func bestMatch(query string, queryYear int, results []tmdb.MovieSummary) (tmdb.MovieSummary, int) {
	var best tmdb.MovieSummary
	bestScore := -1

	for _, r := range results {
		score := scoreMatch(query, queryYear, r)
		if score > bestScore || (score == bestScore && r.VoteCount > best.VoteCount) {
			bestScore = score
			best = r
		}
	}

	return best, bestScore
}

func scoreMatch(query string, queryYear int, result tmdb.MovieSummary) int {
	titleScore := scoreTitleMatch(query, result.Title, result.OriginalTitle)
	yearScore := scoreYearMatch(queryYear, extractReleaseYear(result.ReleaseDate))
	confidenceScore := scoreConfidence(result.VoteCount, result.Popularity)
	return titleScore + yearScore + confidenceScore
}

// scoreTitleMatch scores how well the result title matches the query.
// Returns 0-100.
func scoreTitleMatch(query, title, originalTitle string) int {
	qNorm := normalizeTitle(query)
	tNorm := normalizeTitle(title)
	otNorm := normalizeTitle(originalTitle)

	if qNorm == tNorm {
		return 100
	}
	if qNorm == otNorm {
		return 95
	}

	// Strip punctuation and compare
	qStripped := stripPunctuation(qNorm)
	tStripped := stripPunctuation(tNorm)
	otStripped := stripPunctuation(otNorm)

	if qStripped == tStripped || qStripped == otStripped {
		return 85
	}

	// Check if one is a prefix of the other (handles subtitle truncation,
	// e.g. "Wake Up Dead Man" vs "Wake Up Dead Man: A Knives Out Mystery")
	if isPrefixMatch(qStripped, tStripped) || isPrefixMatch(qStripped, otStripped) {
		return 70
	}

	// Check if one contains the other with high length ratio
	if containsWithHighRatio(qStripped, tStripped) || containsWithHighRatio(qStripped, otStripped) {
		return 60
	}

	return 0
}

// scoreYearMatch scores year proximity. Returns 0-50.
func scoreYearMatch(queryYear, resultYear int) int {
	if resultYear == 0 || queryYear == 0 {
		return 0
	}
	diff := queryYear - resultYear
	if diff < 0 {
		diff = -diff
	}
	switch diff {
	case 0:
		return 50
	case 1:
		return 30
	case 2:
		return 10
	default:
		return 0
	}
}

// scoreConfidence adds a small tiebreaker based on vote count and popularity.
// Returns 0-10.
func scoreConfidence(voteCount int, popularity float64) int {
	score := 0
	if voteCount >= 100 {
		score += 5
	}
	if popularity >= 10.0 {
		score += 5
	}
	return score
}

func extractReleaseYear(releaseDate string) int {
	if len(releaseDate) < 4 {
		return 0
	}
	year, err := strconv.Atoi(releaseDate[:4])
	if err != nil {
		return 0
	}
	return year
}

func normalizeTitle(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func stripPunctuation(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// isPrefixMatch checks if one string is a word-boundary prefix of the other
// AND covers a significant portion of the full title (ratio > 0.3).
// Handles subtitle truncation (e.g. "Glass Onion" is a prefix of
// "Glass Onion A Knives Out Mystery" — ratio 0.34, passes).
// Rejects short-word false positives (e.g. "Dolly" prefix of
// "Dolly Parton LAmérique réconciliée" — ratio 0.16, rejected;
// "Guru" prefix of "Guru Nanak Jahaz" — ratio 0.27, rejected).
func isPrefixMatch(a, b string) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}

	shorter, longer := a, b
	if len(a) > len(b) {
		shorter, longer = b, a
	}

	if !strings.HasPrefix(longer, shorter) {
		return false
	}

	// Ensure the prefix ends at a word boundary (next char must be a space)
	if len(longer) > len(shorter) && longer[len(shorter)] != ' ' {
		return false
	}

	// Require the prefix to cover a meaningful portion of the full title
	ratio := float64(len(shorter)) / float64(len(longer))
	return ratio > 0.3
}

// containsWithHighRatio checks if one string contains the other
// and they are similar in length (ratio > 0.7).
func containsWithHighRatio(a, b string) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}

	shorter, longer := a, b
	if len(a) > len(b) {
		shorter, longer = b, a
	}

	if !strings.Contains(longer, shorter) {
		return false
	}

	ratio := float64(len(shorter)) / float64(len(longer))
	return ratio > 0.7
}
