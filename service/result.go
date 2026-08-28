package service

// Status describes the outcome of one service operation.
type Status string

const (
	StatusProcessed Status = "processed"
	StatusWritten   Status = "written"
	StatusSkipped   Status = "skipped"
	StatusFailed    Status = "failed"
)
