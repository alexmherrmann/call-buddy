package telephono

import (
	"encoding/json"
	"errors"
	"strings"
)

type HttpMethod string

const (
	Post   HttpMethod = "POST"
	Get               = "GET"
	Put               = "PUT"
	Delete            = "DELETE"
	Head              = "HEAD"
)

// FIXME DG? Is this necessary?
func (m *HttpMethod) UnmarshalJSON(buf []byte) error {
	var method string
	if err := json.Unmarshal(buf, &method); err != nil {
		return err
	}
	walrus, err := toHttpMethod(method)
	*m = walrus
	return err
}

func (m HttpMethod) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.String())
}

func AllHttpMethods() []HttpMethod {
	return []HttpMethod{Post, Get, Put, Delete, Head}
}

func (m HttpMethod) String() string {
	return string(m)
}

func toHttpMethod(method string) (HttpMethod, error) {
	methodUpper := strings.ToUpper(method)
	switch methodUpper {
	case "POST":
		return Post, nil
	case "GET":
		return Get, nil
	case "PUT":
		return Put, nil
	case "DELETE":
		return Delete, nil
	case "HEAD":
		return Head, nil
	}
	return "", errors.New("No such HTTP method " + method)
}
