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
// If IncludeColumnInURLWildcardSegment=true, it only excludes '/'.
func singleWildcardReplacementRegex(IncludeColumnInURLWildcardSegment bool) string {
	if IncludeColumnInURLWildcardSegment {
		return `[^\/]+`
	}
	return `[^\/:]+`
}

// Wildcard segment matching any char 0 or unlimited times, except ':'.
// If IncludeColumnInURLWildcardSegment=true, it matches any char.
func doubleWildcardReplacementRegex(IncludeColumnInURLWildcardSegment bool) string {
	if IncludeColumnInURLWildcardSegment {
		return `.*`
	}
	return `[^:]*`
}
