package service

// Status describes the outcome of one service operation.
type Status string

const (
	StatusProcessed Status = "processed"
	StatusWritten   Status = "written"
	StatusSkipped   Status = "skipped"
	StatusFailed    Status = "failed"
)

// ItemResult records the outcome for one object in a bulk operation.
type ItemResult struct {
	ObjectID string
	Status   Status
	Err      error
}

// BatchResult summarizes a bulk operation while retaining item-level errors.
type BatchResult struct {
	Total     int
	Succeeded int
	Skipped   int
	Failed    int
	Items     []ItemResult
}

func newBatchResult(total int) BatchResult {
	return BatchResult{
		Total: total,
		Items: make([]ItemResult, 0, total),
	}
}

func (r *BatchResult) add(objectID string, status Status, err error) {
	if err != nil {
		status = StatusFailed
	}

	r.Items = append(r.Items, ItemResult{
		ObjectID: objectID,
		Status:   status,
		Err:      err,
	})

	switch status {
	case StatusSkipped:
		r.Skipped++
	case StatusFailed:
		r.Failed++
	default:
		r.Succeeded++
	}
}
