package telephono

import (
	"net/http"
	"strings"
)

type Body BasicExpandable

type RequestTemplate struct {
	Method         HttpMethod
	Url            *BasicExpandable
	Headers        HeadersTemplate
	ExpandableBody *BasicExpandable
	// TODO AH: specify a body type that's just given a reader.
}

//executeWithClientAndExpander will execute this call template with the specified client and expander, returning a response or an error
func (r *RequestTemplate) ExecuteWithClientAndExpander(client *http.Client, expander Expander) (HistoricalCall, error) {
	//expand the url
	expandedUrl, urlErr := r.Url.Expand(expander)
	if urlErr != nil {
		return HistoricalCall{}, urlErr
	}

	//expand the body
	//TODO AH: file bodies for things like binary data or purposefully unrendered stuff
	//OPTIMIZE AH: Instead of just expanding this, stream it so that we're not loading so many things into memory
	expandedBody, bodyErr := r.ExpandableBody.Expand(expander)
	if bodyErr != nil {
		return HistoricalCall{}, bodyErr
	}

	bodyReader := strings.NewReader(expandedBody)
	httpRequest, newCallErr := http.NewRequestWithContext(globalState.callContext, string(r.Method), expandedUrl, bodyReader)
	if newCallErr != nil {
		return HistoricalCall{}, newCallErr
	}

	// This must be done before we do our call since the call consumes the body
	request := Request{}
	request.Populate(httpRequest)

	// Add the headers
	if header, errors := r.Headers.ExpandAllAsHeader(expander); len(errors) == 0 {
		httpRequest.Header = header
	}

	httpResponse, doErr := client.Do(httpRequest)
	if doErr != nil {
		return HistoricalCall{}, doErr
	}

	response := Response{}
	response.Populate(httpResponse)

	call := HistoricalCall{Request: request, Response: response}
	return call, nil
}
