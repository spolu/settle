package mint

// TkName represents a task name.
type TkName string

// TkStatus represents a task status.
type TkStatus string

const (
	// TkStPending new or have been retried less than the task max retries.
	TkStPending TkStatus = "pending"
	// TkStSucceeded successfully executed once.
	TkStSucceeded TkStatus = "succeeded"
	// TkStFailed retried more than max retries with no success.
	TkStFailed TkStatus = "failed"
)
