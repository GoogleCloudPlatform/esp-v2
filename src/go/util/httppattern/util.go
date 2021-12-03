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
// If ExcludeColonInUrlWildcardPathSegment=true, it excludes '/' and ':'.
func singleWildcardReplacementRegex(ExcludeColonInUrlWildcardPathSegment bool) string {
	if ExcludeColonInUrlWildcardPathSegment {
		return `[^\/:]+`
	}
	return `[^\/]+`
}

// Wildcard segment matching any char 0 or unlimited times, except ':'.
// If ExcludeColonInUrlWildcardPathSegment=true, it matches any char.
func doubleWildcardReplacementRegex(ExcludeColonInUrlWildcardPathSegment bool) string {
	if ExcludeColonInUrlWildcardPathSegment {
		return `[^:]*`
	}
	return `.*`
}
