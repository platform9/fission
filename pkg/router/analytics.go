package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/dchest/uniuri"
)

type (
	// Analytics helps exporting usage metrics
	Analytics struct {
		id  string
		url string
	}

	// AnalyticsData is the data exported as usage metrics
	AnalyticsData struct {
		ID                string `json:"ID"`
		FunctionCallCount uint64 `json:"FunctionCallCount"`
	}
)

// MakeAnalytics returns a new instance of Analytics if url or
// 'ANALYTICS_URL' environment variable is set; nil otherwise
func MakeAnalytics(url string) *Analytics {
	if len(url) == 0 {
		url = os.Getenv("ANALYTICS_URL")
		if len(url) == 0 {
			return nil
		}
	}

	a := &Analytics{
		url: url,
		id:  uniuri.NewLen(8),
	}
	go a.run()
	return a
}

func (a *Analytics) gatherData() *AnalyticsData {
	return &AnalyticsData{
		FunctionCallCount: atomic.LoadUint64(&globalFunctionCallCount),
	}
}

func (a *Analytics) run() {
	ticker := time.NewTicker(24 * time.Hour)
	for range ticker.C {
		msg := a.gatherData()
		msg.ID = a.id

		msgbytes, err := json.Marshal(*msg)
		if err != nil {
			continue
		}

		resp, _ := http.Post(a.url, "application/json", bytes.NewReader(msgbytes))
		if resp != nil {
			// close response body to prevent resources leak
			resp.Body.Close()
		}
	}
}
