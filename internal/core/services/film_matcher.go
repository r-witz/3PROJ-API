package services

import (
	"strconv"
	"strings"
	"unicode"

	"duskforge-api/pkg/tmdb"
)

const matchThreshold = 80

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

	qStripped := stripPunctuation(qNorm)
	tStripped := stripPunctuation(tNorm)
	otStripped := stripPunctuation(otNorm)

	if qStripped == tStripped || qStripped == otStripped {
		return 85
	}

	if isPrefixMatch(qStripped, tStripped) || isPrefixMatch(qStripped, otStripped) {
		return 70
	}

	if containsWithHighRatio(qStripped, tStripped) || containsWithHighRatio(qStripped, otStripped) {
		return 60
	}

	return 0
}

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

	if len(longer) > len(shorter) && longer[len(shorter)] != ' ' {
		return false
	}

	ratio := float64(len(shorter)) / float64(len(longer))
	return ratio > 0.3
}

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
