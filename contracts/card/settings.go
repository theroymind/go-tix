package card

const (
	FieldID         = "id"
	FieldType       = "type"
	FieldTitle      = "title"
	FieldSummary    = "summary"
	FieldHasSummary = "has_summary"
	FieldDistance   = "distance"
	FieldCommits    = "commits"
	FieldFiles      = "files"
)

const (
	minimumDistance        = 0
	problemRequired        = "is required"
	problemEmptyItem       = "must not be empty"
	problemSummaryMismatch = "must be true exactly when the summary is not empty"
)
