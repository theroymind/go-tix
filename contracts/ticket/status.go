package ticket

import "slices"

type Status string

const (
	StatusDraft      Status = "draft"
	StatusOpen       Status = "open"
	StatusInProgress Status = "in_progress"
	StatusCompleted  Status = "completed"
	StatusCanceled   Status = "canceled"
)

var Statuses = []Status{StatusDraft, StatusOpen, StatusInProgress, StatusCompleted, StatusCanceled}

func (status Status) IsValid() bool {
	return slices.Contains(Statuses, status)
}

func (status Status) IsResolved() bool {
	return status == StatusCompleted || status == StatusCanceled
}
