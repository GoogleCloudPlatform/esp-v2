package httppattern

const (
	//  The special segment keys during uri template parsing.
	SingleParameterKey = "/."
	SingleWildCardKey  = "*"
	DoubleWildCardKey  = "**"

	HttpMethodWildCard = "*"

	// Matches 1 or more segments of any character except '/'.
	singleWildcardReplacementRegex = `[^\/]+`
	// Matches any character or no characters at all.
	doubleWildcardReplacementRegex = `.*`
)
