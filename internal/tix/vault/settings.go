package vault

import "regexp"

const (
	frontmatterDelimiter = "---"
	noteExtension        = ".md"
	yamlIndent           = 2
	carriageReturn       = "\r"
	doneMark             = "x"
	numberedMarker       = "."
	undoneMark           = " "
	maxFenceIndent       = 3
	stringTag            = "!!str"
	slugSeparator        = "-"
	headingMark          = "#"
)

var (
	headingPattern    = regexp.MustCompile(`^ {0,3}(#{1,6})(?:[ \t]+(.*?))?(?:[ \t]+#+)?[ \t]*$`)
	fencePattern      = regexp.MustCompile("^ {0,3}(`{3,}|~{3,})")
	listMarkerPattern = regexp.MustCompile(`^(\s*)(?:[-*+]|\d+[.)])(\s+\[)$`)
	checkboxPattern   = regexp.MustCompile(`^(\s*(?:[-*+]|\d+[.)])\s+\[)([ xX])(\](?:[ \t]+|$))(.*)$`)
	wikilinkPattern   = regexp.MustCompile(`!?\[\[([^\[\]\n]*)\]\]`)
	backtickRun       = regexp.MustCompile("`+")
)
