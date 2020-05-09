package gqlws

const (
	// Client -> Server
	MsgTypeConnectionInit      = "connection_init"
	MsgTypeConnectionTerminate = "connection_terminate"
	MsgTypeStart               = "start"
	MsgTypeStop                = "stop"

	// Server -> Client
	MsgTypeConnectionAck       = "connection_ack"
	MsgTypeConnectionError     = "connection_error"
	MsgTypeConnectionKeepAlive = "ka"
	MsgTypeData                = "data"
	MsgTypeError               = "error"
	MsgTypeComplete            = "complete"
)
