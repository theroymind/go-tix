package ticket

import "slices"

type Status string

const (
	StatusDraft      Status = "draft"
	StatusOpen       Status = "open"
	StatusInProgress Status = "in_progress"
	StatusInReview   Status = "in_review"
	StatusCompleted  Status = "completed"
	StatusCanceled   Status = "canceled"
)

var Statuses = []Status{StatusDraft, StatusOpen, StatusInProgress, StatusInReview, StatusCompleted, StatusCanceled}

func (status Status) IsValid() bool {
	return slices.Contains(Statuses, status)
}

func (status Status) IsResolved() bool {
	return status == StatusCompleted || status == StatusCanceled
}
