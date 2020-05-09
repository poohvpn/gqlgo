package gqlgo

import (
	"encoding/json"
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

type GraphQLErrorLocation struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

type DetailError struct {
	OriginError error
	Content     string
	Response    *http.Response
}

func jsonifyError(e interface{}) string {
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
	return jsonifyError(e)
}

func (e *GraphQLError) Error() string {
	return jsonifyError(e)
}

func (e *DetailError) Error() string {
	if e == nil || e.OriginError == nil {
		return "<nil>"
	}
	return e.OriginError.Error()
}

func errFileReaderIsNil(path string) error {
	return errors.Errorf("requests.%s: file reader is required", path)
}
