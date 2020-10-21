package httppattern

import (
	"regexp"
	"strings"
)

//  The special segment keys during uri template parsing.
const (
	SingleParameterKey = "/."
	SingleWildCardKey  = "*"
	DoubleWildCardKey  = "**"
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

// This function return the uri string with snakeNames replaced with jsonName.
// It assume:
//   - the input uri template is valid and it won't verify the uri.
//   - each snakeName as variable in the input uri appear equal to or less than once.
//
// It uses the hacky substring replacement:
//   - find the first appearance of snakeName, the char before which is '{' or '.',
//     the char after which is '}' or '.' or '='
//   - replace that substring with the jsonName
//
// Same replacement cane be expressed as regexReplace(`(?<=[.{])${snakeName}(?=[.}=])`, ${jsonName})
// but golang doesn't support such look around syntax.
//
// It should match the variable name extraction behavior in
// https://github.com/GoogleCloudPlatform/esp-v2/blob/34314a46a54001f83508071e78596cba08b6f456/src/api_proxy/path_matcher/http_template_test.cc
//
// TODO(taoxuy@): extract variable name by syntax parsing.
func SnakeNamesToJsonNamesInPathParam(uri string, snakeNameToJsonName map[string]string) string {
	findPathParamIndex := func(uri, snakeName string) int {
		for {
			index := strings.Index(uri, snakeName)
			if index == -1 {
				return -1
			}

			if index != 0 && index+len(snakeName) < len(uri) {
				// If the leftSide of snakeName match is `{` or '.'.
				leftSide := uri[index-1] == '{' || uri[index-1] == '.'

				// If the rightSide of snakeName match is `}`, '.' or '='.
				rightSide := uri[index+len(snakeName)] == '}' || uri[index+len(snakeName)] == '.' || uri[index+len(snakeName)] == '='

				if leftSide && rightSide {
					return index
				}
			}

			uri = uri[index+len(snakeName):]
			continue
		}
	}

	snakeNameToJsonNameInPathParam := func(uri, snakeName, jsonName string) string {
		index := findPathParamIndex(uri, snakeName)
		if index == -1 {
			return uri
		}

		return uri[0:index] + jsonName + uri[index+len(snakeName):]
	}

	for snakeName, jsonName := range snakeNameToJsonName {
		uri = snakeNameToJsonNameInPathParam(uri, snakeName, jsonName)

	}

	return uri
}
