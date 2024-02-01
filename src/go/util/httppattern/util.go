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

// SingleWildcardReplacementRegex does wildcard segment matching any char 1 or unlimited times, except '/'.
// If disallowColonInWildcardPathSegment=true, it matches any char except '/' and ':'.
func SingleWildcardReplacementRegex(disallowColonInWildcardPathSegment bool) string {
	if disallowColonInWildcardPathSegment {
		return `[^\/:]+`
	}
	return `[^\/]+`
}

// DoubleWildcardReplacementRegex does wildcard segment matching any char 0 or unlimited times, except '/'.
// If disallowColonInWildcardPathSegment=true, it matches any char except '/' and ':'.
func DoubleWildcardReplacementRegex(disallowColonInWildcardPathSegment bool) string {
	if disallowColonInWildcardPathSegment {
		return `[^:]*`
	}
	return `.*`
}
