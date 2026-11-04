package card

const (
	FieldID         = "id"
	FieldType       = "type"
	FieldTitle      = "title"
	FieldSummary    = "summary"
	FieldHasSummary = "has_summary"
	FieldDistance   = "distance"
	FieldRelation   = "relation"
	FieldCommits    = "commits"
	FieldFiles      = "files"
	FieldScore      = "score"
)

const (
	RelationStart      = "start"
	RelationParent     = "parent"
	RelationDependency = "dependency"
	RelationWikilink   = "wikilink"
	RelationBacklink   = "backlink"
)

const (
	minimumDistance        = 0
	minimumScore           = -1.0
	maximumScore           = 1.0
	problemScoreRange      = "must be between -1 and 1"
	problemRelationUnknown = "must be start, parent, dependency, wikilink or backlink"
	problemStartMismatch   = "must be start exactly at distance 0"
	problemRequired        = "is required"
	problemEmptyItem       = "must not be empty"
	problemSummaryMismatch = "must be true exactly when the summary is not empty"
)
