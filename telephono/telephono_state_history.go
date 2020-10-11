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
		CallsFromCurrentSession []HistoricalCall
	}
)

func (wholeHistory *CallBuddyHistory) AddFinishedCall(call HistoricalCall) {
	wholeHistory.CallsFromCurrentSession = append(wholeHistory.CallsFromCurrentSession, call)
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

	for _, call := range wholeHistory.CallsFromCurrentSession {
		buffer.WriteString(call.GetSimpleReport())
		// HMM AH: Multiplatform
		buffer.WriteByte('\n')
	}

	return buffer.String()
}

// ^ Same comment probably applies here -Dylan
func (wholeHistory *CallBuddyHistory) GetNthCommand(n int) (string, error) {
	if n < 0 || n > len(wholeHistory.CallsFromCurrentSession)-1 {
		return "", fmt.Errorf("No command at pos %d", n)
	}
	call := wholeHistory.CallsFromCurrentSession[n]

	// {method} {request URL} [content-type]
	cmd := fmt.Sprintf("%s %s", call.Request.Method, call.Request.URL)

	// Golang's net.http.Header is a map[string][]string for some reason
	if len(call.Request.Header["Content-type"]) > 0 {
		cmd += " " + call.Request.Header["Content-type"][0]
	}
	return cmd, nil
}

func (wholeHistory *CallBuddyHistory) Size() int {
	return len(wholeHistory.CallsFromCurrentSession)
}
