package ticket

import "slices"

type Type string

const (
	TypeEpic  Type = "epic"
	TypeStory Type = "story"
	TypeTask  Type = "task"
)

var Types = []Type{TypeEpic, TypeStory, TypeTask}

func (ticketType Type) IsValid() bool {
	return slices.Contains(Types, ticketType)
}
