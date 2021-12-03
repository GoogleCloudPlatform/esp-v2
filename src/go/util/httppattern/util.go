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
// If allowColonInWildcardPathSegment=true, it matches any char except '/'.
func singleWildcardReplacementRegex(allowColonInWildcardPathSegment bool) string {
	if allowColonInWildcardPathSegment {
		return `[^\/]+`
	}
	return `[^\/:]+`
}

// Wildcard segment matching any char 0 or unlimited times, except '/' and ':'.
// If allowColonInWildcardPathSegment=true, it matches any char except '/'.
func doubleWildcardReplacementRegex(allowColonInWildcardPathSegment bool) string {
	if allowColonInWildcardPathSegment {
		return `.*`
	}
	return `[^:]*`
}
