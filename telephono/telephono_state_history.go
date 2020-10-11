package telephono

import (
	"fmt"
	"strings"
)

type (
	//SimpleReportable interface {
	//	GetSimpleReport() string
	//}

	HistoricalCall struct {
		Response Response
		Request  Request
	}

	CallBuddyHistory struct {
		callsFromCurrentSession []HistoricalCall
	}
)

func (wholeHistory *CallBuddyHistory) AddFinishedCall(call HistoricalCall) {
	wholeHistory.callsFromCurrentSession = append(wholeHistory.callsFromCurrentSession, call)
}

// GetSimpleReport generates simple string report that gives info about the request/response
func (theCall HistoricalCall) GetSimpleReport() string {
	// {method} {request URL}: {response code} [content length]
	return fmt.Sprintf("%8s %-50s: [%3d] [%5d] bytes", theCall.Request.Method, theCall.Request.URL, theCall.Response.StatusCode, len(theCall.Response.Body))
}

// TODO AH: May not be this method's concern, but this is hacky and will get big quickly
// GetSimpleWholeHistoryReport Generates a big string of all the calls
func (wholeHistory *CallBuddyHistory) GetSimpleWholeHistoryReport() string {
	buffer := strings.Builder{}

	for _, call := range wholeHistory.callsFromCurrentSession {
		buffer.WriteString(call.GetSimpleReport())
		// HMM AH: Multiplatform
		buffer.WriteByte('\n')
	}

	return buffer.String()
}

func (wholeHistory *CallBuddyHistory) Get(n int) (HistoricalCall, error) {
	if n < 0 || n > len(wholeHistory.callsFromCurrentSession)-1 {
		return HistoricalCall{}, fmt.Errorf("No history at pos %d", n)
	}
	return wholeHistory.callsFromCurrentSession[n], nil
}

func (wholeHistory *CallBuddyHistory) Size() int {
	return len(wholeHistory.callsFromCurrentSession)
}
