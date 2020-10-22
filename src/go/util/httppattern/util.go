package httppattern

import (
	"regexp"
)

const (
	//  The special segment keys during uri template parsing.
	SingleParameterKey = "/."
	SingleWildCardKey  = "*"
	DoubleWildCardKey  = "**"

	HttpMethodWildCard = "*"
)

var (
	// Various hacky regular expressions to match a subset of the http template syntax.

	// Match and capture the segment binding for a named field path.
	// - /v1/{resource=shelves/*/books/**} -> /v1/shelves/*/books/**
	fieldPathSegmentSimplifier = regexp.MustCompile(`{[^{}]+=([^{}]+)}`)
	// Replace segments with single wildcards
	// - /v1/books/* -> /v1/books/[^/]+
	singleWildcardMatcher = regexp.MustCompile(`/\*`)
	// Replace segments with double wildcards
	// - /v1/** -> /v1/.*
	doubleWildcardMatcher = regexp.MustCompile(`/\*\*`)
	// Replace any path templates
	// - /v1/books/{book_id} -> /v1/books/[^/]+
	pathParamMatcher = regexp.MustCompile(`/{[^{}]+}`)

	// Common regex forms that emulate http template syntax.

	// Matches 1 or more segments of any character except '/'.
	singleWildcardReplacementRegex = `/[^\/]+`
	// Matches any character or no characters at all.
	doubleWildcardReplacementRegex = `/.*`
)

// Returns a regex that will match requests to the uri with path parameters or wildcards.
// If there are no path params or wildcards, returns empty string.
//
// Essentially matches a subset of the http template syntax.
// FIXME(nareddyt): Remove this hack completely when envoy route config supports path matching with path templates.
func WildcardMatcherForPath(uri string) string {

	// Ordering matters, start with most specific and work upwards.
	matcher := fieldPathSegmentSimplifier.ReplaceAllString(uri, "$1")
	matcher = pathParamMatcher.ReplaceAllString(matcher, singleWildcardReplacementRegex)
	matcher = doubleWildcardMatcher.ReplaceAllString(matcher, doubleWildcardReplacementRegex)
	matcher = singleWildcardMatcher.ReplaceAllString(matcher, singleWildcardReplacementRegex)

	if matcher == uri {
		return ""
	}

	// Enforce strict prefix / suffix.
	return "^" + matcher + "$"
}
