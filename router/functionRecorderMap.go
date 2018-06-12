package router

type (
	functionRecorderMap map[string]bool
)

/*
type Record struct {
	recordFunctionMapping map[string]bool
	recordTriggerMapping  map[string]bool
}

// Method or function?
func (r *Record) recordFunction(name string) bool {
	return r.recordFunctionMapping[name]
}

func (r *Record) recordTrigger(name string) bool {
	return r.recordTriggerMapping[name]
}
*/