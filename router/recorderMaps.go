package router

type (
	FunctionRecorderMap map[string]bool
	TriggerRecorderMap map[string]bool
)

// Very simple at the moment
func MakeFunctionRecorderMap() *FunctionRecorderMap {
	var frmap FunctionRecorderMap
	return &frmap
}

func MakeTriggerRecorderMap() *TriggerRecorderMap {
	var trmap TriggerRecorderMap
	return &trmap
}

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