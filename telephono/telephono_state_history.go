package telephono

import (
	"fmt"
	"net/http"
	"strings"
)

type (
	//SimpleReportable interface {
	//	GetSimpleReport() string
	//}

	HistoricalCall struct {
		request  http.Request
		response http.Response
	}

	CallBuddyHistory struct {
		callsFromCurrentSession []HistoricalCall
	}
)

func (wholeHistory *CallBuddyHistory) AddFinishedCall(input CallResponse) {
	wholeHistory.callsFromCurrentSession = append(wholeHistory.callsFromCurrentSession, HistoricalCall{
		request:  *(input.Request),
		response: *input,
	})
}

// GetSimpleReport generates simple string report that gives info about the request/response
func (theCall HistoricalCall) GetSimpleReport() string {
	// {method} {request URL}: {response code} [content length]
	return fmt.Sprintf("%8s %-50s: [%3d] [%5d] bytes", theCall.request.Method, theCall.request.URL.String(), theCall.response.StatusCode, theCall.response.ContentLength)
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
