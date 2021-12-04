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

// Wildcard segment matching any char 1 or unlimited times, except '/'.
// If disallowColonInWildcardPathSegment=true, it matches any char except '/' and ':'.
func singleWildcardReplacementRegex(disallowColonInWildcardPathSegment bool) string {
	if disallowColonInWildcardPathSegment {
		return `[^\/:]+`
	}
	return `[^\/]+`
}

// Wildcard segment matching any char 0 or unlimited times, except '/'.
// If disallowColonInWildcardPathSegment=true, it matches any char except '/' and ':'.
func doubleWildcardReplacementRegex(disallowColonInWildcardPathSegment bool) string {
	if disallowColonInWildcardPathSegment {
		return `[^:]*`
	}
	return `.*`
}
