package gqlgo

import (
	"io"
	"net/http"
)

// Option be changed at anytime after NewClient
type Option struct {
	// Endpoint means server URL
	Endpoint string

	// HTTPClient specify http client, when it's nil, GraphQL client will use http.DefaultClient
	// HTTPClient should not change to nil after init
	HTTPClient *http.Client

	// Headers appended to http request every time at the beginning
	Headers map[string]string

	// CloseBody will close http request body immediately for reusing of http client
	CloseBody bool

	// Client will add Header "Authorization: Bearer <Token>" for every request when BearerAuth is not empty
	BearerAuth string

	// Custom HTTP Log func like func(s string) { fmt.Println(s) }
	Log func(msg string)

	// NotCheckHTTPStatusCode200 disable http response status code for some irregular GraphQL Servers
	NotCheckHTTPStatusCode200 bool
}

type Client struct {
	*Option
}

type Request struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

type File struct {
	Reader io.Reader
	Name   string
}
