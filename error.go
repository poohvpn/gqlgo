package gqlgo

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
)

type GraphQLErrors []GraphQLError

// Path element's type should be either string or int, according to the samples of http://spec.graphql.org/draft/#sec-Errors
type GraphQLError struct {
	Message    string                 `json:"message,omitempty"`
	Locations  []GraphQLErrorLocation `json:"locations,omitempty"`
	Path       []interface{}          `json:"path,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

type HTTPError struct {
	Response  *http.Response
	SavedBody string
}

type GraphQLErrorLocation struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

type JsonError struct {
	OriginError error
	Json        string
}

func JsonifyError(e interface{}) string {
	if e == nil {
		return "null"
	}
	j, err := json.Marshal(e)
	if err != nil {
		return err.Error()
	}
	return string(j)
}

func (e GraphQLErrors) Error() string {
	return JsonifyError(e)
}

func (e *GraphQLError) Error() string {
	return JsonifyError(e)
}

func (e *HTTPError) Error() string {
	if e == nil || e.Response == nil {
		return "<nil>"
	}
	return fmt.Sprintf("unexpected HTTP response code: %d", e.Response.StatusCode)
}

func (e *JsonError) Error() string {
	if e == nil || e.OriginError == nil {
		return "<nil>"
	}
	return e.OriginError.Error()
}

func errFileReaderIsNil(path string) error {
	return errors.Errorf("requests.%s: file reader is required", path)
}
