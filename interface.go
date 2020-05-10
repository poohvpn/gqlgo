package gqlgo

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// Option be changed at anytime after NewClient
type Option struct {
	// Endpoint means HTTP server URL
	Endpoint string

	// HTTPClient specify http client, when it's nil, GraphQL client will use http.DefaultClient
	// HTTPClient should not change to nil after init
	HTTPClient *http.Client

	// Headers apply to http request every time at the beginning
	Headers map[string]string

	// CloseBody will close http request body immediately for reusing of http client
	CloseBody bool

	// Client will add Header "Authorization: Bearer <Token>" for every request when BearerAuth is not empty
	BearerAuth string

	// Custom HTTP Log func like func(s string) { fmt.Println(s) }
	Log func(msg string)

	// NotCheckHTTPStatusCode200 disable http response status code for some irregular GraphQL Servers
	NotCheckHTTPStatusCode200 bool

	// WebSocketEndpoint specify websocket endpoint, default is Endpoint's websocket schema
	WebSocketEndpoint string

	WebSocketOption WSOption
}

type Request struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables"`
	OperationName string                 `json:"operationName,omitempty"`
	Extensions    interface{}            `json:"extensions,omitempty"`

	// Headers apply to http request at last
	Headers map[string]string `json:"-"`
}

type File struct {
	Reader io.Reader
	Name   string
}

// Option be changed at anytime after NewWSClient
type WSOption struct {
	// Dialer specify websocket Dialer, default is using websocket.DefaultDialer
	Dialer *websocket.Dialer

	Headers map[string]string

	NotReconnect bool

	// ReconnectAttempts is the maximum attempts of reconnection after connected, default is math.MaxUint32
	ReconnectAttempts uint32

	// KeepAliveTimeout is the timeout of server keepalive since last keepalive message.
	// Default is 30 seconds, less or equal than 10 second will disable checking keepalive timeout.
	KeepAliveTimeout time.Duration

	// Custom WebSocket GraphQL Log func like func(s string) { fmt.Println(s) }
	Log func(msg string)
}

// GQL_ERROR will be appended to errors, then errors will be a list that contains only one error.
// completed is true only happens to GraphQL server send completed, if completed is true, data and errors must be nil.
// while returned error is not nil, Subscription will be unsubscribed.
type SubscriptionHandler func(data json.RawMessage, errors GraphQLErrors, completed bool) error
