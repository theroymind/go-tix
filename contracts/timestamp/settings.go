package timestamp

const (
	Layout         = "2006-01-02T15:04:05-07:00"
	pattern        = `^\d{4}-\d{2}-\d{2}T(?:[01]\d|2[0-3]):[0-5]\d:[0-5]\d(?:\.\d+)?(?:Z|[+-](?:[01]\d|2[0-3]):[0-5]\d)$`
	yamlTag        = "!!timestamp"
	ProblemInvalid = "Invalid ISO datetime"
)
