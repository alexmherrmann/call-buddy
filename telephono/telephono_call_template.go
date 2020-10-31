package telephono

import (
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type RequestTemplate struct {
	Method  HttpMethod
	Url     string
	Headers http.Header
	Body    string // FIXME DG: byte buffer or reader?
}

//executeWithClientAndExpander will execute this call template with the specified client and expander, returning a response or an error
func (r *RequestTemplate) Execute(client *http.Client, env *CallBuddyEnvironment) (HistoricalCall, error) {
	expandedUrl := env.Expand(r.Url)

	// Weird dance where Go wants a body reader for HTTP calls
	method := string(r.Method)
	expandedBody := env.OS.Expand(env.User.Expand(r.Body))
	log.Printf("Body: %s\n", expandedBody)

	if expandedBody == "\n" {
		expandedBody = ""
	}
	bodyReader := strings.NewReader(expandedBody)
	httpRequest, newCallErr := http.NewRequest(method, expandedUrl, bodyReader)
	if newCallErr != nil {
		return HistoricalCall{}, newCallErr
	}

	// This must be done before we do our call since the call consumes the body (since it's a reader)
	// Populate our own structs with Go's http.Request
	request := Request{}
	request.Populate(httpRequest, expandedBody)

	// Add the headers
	header := http.Header{}
	for key, values := range r.Headers {
		for _, value := range values {
			header[key] = append(header[key], env.Expand(value))
		}
	}

	// If the user forgets to add the content-type, we'll be nice and add it for them!
	if method == "POST" || method == "PUT" {
		contentType := header.Get("Content-type")
		if contentType == "" {
			httpRequest.Header["Content-type"] = []string{"text/plain"}
		}
		contentLength := len(expandedBody)
		httpRequest.Header["Content-length"] = []string{strconv.Itoa(contentLength)}
		httpRequest.Body = ioutil.NopCloser(strings.NewReader(expandedBody))
	}
	httpRequest.Header = header

	// Call!
	httpResponse, doErr := client.Do(httpRequest)
	if doErr != nil {
		return HistoricalCall{}, doErr
	}

	// Populate our own structs with Go's http.Response
	response := Response{}
	response.Populate(httpResponse)

	call := HistoricalCall{Request: request, Response: response}
	return call, nil
}
