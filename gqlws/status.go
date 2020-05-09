package gqlws

type Status int

const (
	StatusInitial Status = iota
	StatusConnecting
	StatusOpen
	StatusReconnecting
	StatusClosed
)
