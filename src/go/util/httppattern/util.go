package httppattern

const (
	//  The special segment keys during uri template parsing.
	SingleParameterKey = "/."
	SingleWildCardKey  = "*"
	DoubleWildCardKey  = "**"

	HttpMethodWildCard = "*"

	// Matches a trailing slash at the end of a path.
	optionalTrailingSlashRegex = `\/?`
)

// Wildcard segment matching any char 1 or unlimited times, except '/' and ':'.
// If includeColonInWildcardPathSegment=true, it matches any char except '/'.
func singleWildcardReplacementRegex(includeColonInWildcardPathSegment bool) string {
	if includeColonInWildcardPathSegment {
		return `[^\/]+`
	}
	return `[^\/:]+`
}

// Wildcard segment matching any char 0 or unlimited times, except '/' and ':'.
// If includeColonInWildcardPathSegment=true, it matches any char except '/'.
func doubleWildcardReplacementRegex(includeColonInWildcardPathSegment bool) string {
	if includeColonInWildcardPathSegment {
		return `.*`
	}
	return `[^:]*`
}
