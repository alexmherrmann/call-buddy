package telephono

import (
	"errors"
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

// ^ Same comment probably applies here -Dylan
func (wholeHistory *CallBuddyHistory) GetLastCommand() (string, error) {
	if len(wholeHistory.callsFromCurrentSession) < 0 {
		return "", errors.New("No call history")
	}
	lastCall := wholeHistory.callsFromCurrentSession[0]

	// {method} {request URL} [content-type]
	cmd := fmt.Sprintf("%s %s", lastCall.request.Method, lastCall.request.URL.String())

	// Golang's net.http.Header is a map[string][]string for some reason
	if len(lastCall.request.Header["Content-type"]) > 0 {
		cmd += " " + lastCall.request.Header["Content-type"][0]
	}
	return cmd, nil
}

func (wholeHistory *CallBuddyHistory) Size() int {
	return len(wholeHistory.callsFromCurrentSession)
}
