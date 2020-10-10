package telephono

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type (
	// Override Go's http.Request with permanent body and only
	// the things we care about
	Request struct {
		Method HttpMethod
		URL    string
		Header http.Header
		Body   []byte
	}

	// Override Go's http.Response with permanent body and only
	// the things we care about
	Response struct {
		Status     string
		StatusCode int
		Header     http.Header
		Body       []byte
	}
)

func (request *Request) Populate(httpRequest *http.Request) error {
	method, err := toHttpMethod(httpRequest.Method)
	if err != nil {
		return err
	}
	request.Method = method
	request.URL = httpRequest.URL.String()
	request.Header = httpRequest.Header

	request.Body = []byte("")
	bodyBuffer, err := ioutil.ReadAll(httpRequest.Body)
	if err != nil {
		return err
	}
	httpRequest.Body.Close()
	request.Body = bodyBuffer
	return nil
}

func (response *Response) Populate(httpResponse *http.Response) error {
	response.Status = httpResponse.Status
	response.StatusCode = httpResponse.StatusCode
	response.Header = httpResponse.Header

	response.Body = []byte("")
	bodyBuffer, err := ioutil.ReadAll(httpResponse.Body)
	if err != nil {
		return err
	}
	httpResponse.Body.Close()
	response.Body = bodyBuffer
	return nil
}

func (response *Response) String() (result string) {
	if len(response.Header) > 1 {
		for key, value := range response.Header {
			result += fmt.Sprintf("%s: %s\n", key, strings.Trim(strings.Join(value, " "), "[]"))
		}
		result += "\n"
	}
	result += string(response.Body)
	return
}
