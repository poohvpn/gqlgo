package gqlgo

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jpillora/backoff"
	"github.com/pkg/errors"
	"github.com/poohvpn/gqlgo/gqlws"
)

type WSClient struct {
	*WSOption

	endpoint          string
	conn              *websocket.Conn
	id                int64
	subs              sync.Map
	status            gqlws.Status
	unsentRawMsgQueue [][]byte
	lastKA            time.Time
	reconnectBackoff  *backoff.Backoff
	msgWriteMutex     sync.Mutex
}

func NewWSClient(endpoint string, opt ...WSOption) *WSClient {
	client := &WSClient{
		WSOption: &WSOption{},
		reconnectBackoff: &backoff.Backoff{
			Factor: 1.5,
			Min:    time.Second,
			Max:    30 * time.Second,
		},
	}
	if len(opt) > 0 {
		client.WSOption = &opt[0]
	}
	client.endpoint = endpoint
	if client.Dialer == nil {
		client.Dialer = websocket.DefaultDialer
	}
	if client.ReconnectAttempts == 0 {
		client.ReconnectAttempts = math.MaxUint32
	}
	if client.KeepAliveTimeout == 0 {
		client.KeepAliveTimeout = time.Second * 30
	}
	return client
}

func (c *WSClient) Subscribe(req Request, handler SubscriptionHandler) (id string, err error) {
	id = fmt.Sprint(atomic.AddInt64(&c.id, 1))
	err = c.sendMessage(gqlws.MsgTypeStart, id, req)
	if err != nil {
		return
	}
	c.subs.Store(id, handler)
	return
}

func (c *WSClient) Unsubscribe(id string) error {
	if _, ok := c.subs.Load(id); ok {
		c.subs.Delete(id)
		return c.sendMessage(gqlws.MsgTypeStop, id, nil)
	}
	return nil
}

func (c *WSClient) UnsubscribeAll() error {
	var errs []error
	c.subs.Range(func(id, value interface{}) bool {
		c.subs.Delete(id)
		if err := c.sendMessage(gqlws.MsgTypeStop, id.(string), nil); err != nil {
			errs = append(errs, err)
		}
		return true
	})
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

func (c *WSClient) connect() error {
	var (
		httpResp *http.Response
		err      error
	)
	httpHeaders := make(http.Header)
	for k, v := range c.Headers {
		httpHeaders.Set(k, v)
	}
	c.conn, httpResp, err = c.Dialer.Dial(c.endpoint, httpHeaders)
	if err != nil {
		var savedBody []byte
		if httpResp != nil && httpResp.Body != nil {
			savedBody, _ = ioutil.ReadAll(httpResp.Body)
		}
		return &DetailError{
			OriginError: err,
			Response:    httpResp,
			Content:     string(savedBody),
		}
	}

	c.status = gqlws.StatusOpen
	j, _ := json.Marshal(&gqlws.Message{
		Type: gqlws.MsgTypeConnectionInit,
		Payload: struct {
			Headers map[string]string `json:"headers"`
		}{
			Headers: map[string]string{
				"content-type": "application/json",
			},
		},
	})
	go c.run()

	_ = c.sendRawMessage(j)
	c.flushUnsentMessage()

	return nil
}

func (c *WSClient) sendMessage(typ, id string, payload interface{}) error {
	j, err := json.Marshal(&gqlws.Message{
		Type:    typ,
		ID:      id,
		Payload: payload,
	})
	if err != nil {
		return err
	}
	switch c.status {
	case gqlws.StatusInitial:
		c.status = gqlws.StatusConnecting
		c.unsentRawMsgQueue = append(c.unsentRawMsgQueue, j)
		err = c.connect()
		if err != nil {
			return err
		}
		return nil
	case gqlws.StatusConnecting, gqlws.StatusReconnecting:
		c.unsentRawMsgQueue = append(c.unsentRawMsgQueue, j)
		return nil
	case gqlws.StatusOpen:
		return c.sendRawMessage(j)
	default:
		return errors.New("a message was not sent because graphql websocket client is already closed")
	}
}

func (c *WSClient) sendRawMessage(b []byte) error {
	c.msgWriteMutex.Lock()
	defer c.msgWriteMutex.Unlock()
	if c.Log != nil {
		c.Log("send " + string(b))
	}
	w, err := c.conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return err
	}
	defer w.Close()
	_, err = w.Write(b)
	return err
}

func (c *WSClient) run() {
	for {
		if c.KeepAliveTimeout > time.Second*10 &&
			c.lastKA != (time.Time{}) &&
			time.Now().After(c.lastKA.Add(c.KeepAliveTimeout)) {
			c.reconnect()
			return
		}
		msg := gqlws.ResponseMessage{}
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			c.reconnect()
			return
		}
		if c.Log != nil {
			j, _ := json.Marshal(msg)
			c.Log("recv " + string(j))
		}
		switch msg.Type {
		case gqlws.MsgTypeConnectionError:
		case gqlws.MsgTypeConnectionAck:
		case gqlws.MsgTypeConnectionKeepAlive:
			c.lastKA = time.Now()
		case gqlws.MsgTypeComplete:
			if h, ok := c.subs.Load(msg.ID); ok {
				if h != nil {
					_ = h.(SubscriptionHandler)(nil, nil, true)
				}
				c.subs.Delete(msg.ID)
			} else {
				_ = c.sendMessage(gqlws.MsgTypeStop, msg.ID, nil)
			}
		case gqlws.MsgTypeError:
			if h, ok := c.subs.Load(msg.ID); ok {
				if h != nil {
					_ = h.(SubscriptionHandler)(nil, GraphQLErrors{{Message: string(msg.Payload)}}, false)
				}
				c.subs.Delete(msg.ID)
			} else {
				_ = c.sendMessage(gqlws.MsgTypeStop, msg.ID, nil)
			}
		case gqlws.MsgTypeData:
			if h, ok := c.subs.Load(msg.ID); ok {
				if h != nil {
					resp := rawResponse{}
					if err := json.Unmarshal(msg.Payload, &resp); err != nil {
						continue
					}
					var errs GraphQLErrors
					if len(resp.Errors) > 0 {
						errs = resp.Errors
					}
					stopErr := h.(SubscriptionHandler)(resp.Data, errs, false)
					if stopErr != nil {
						_ = c.Unsubscribe(msg.ID)
					}
				}
			} else {
				_ = c.sendMessage(gqlws.MsgTypeStop, msg.ID, nil)
			}
		}
	}
}

func (c *WSClient) Close() error {
	if c.status != gqlws.StatusClosed && c.conn != nil {
		if c.Log != nil {
			c.Log("closing")
		}
		var err error
		err = c.UnsubscribeAll()
		if err != nil {
			return err
		}
		err = c.conn.Close()
		if err != nil {
			return err
		}
		c.status = gqlws.StatusClosed
		c.conn = nil
		return nil
	}
	return nil
}

func (c *WSClient) reconnect() {
	_ = c.Close()
	if c.NotReconnect {
		return
	}

	if c.Log != nil {
		c.Log("reconnecting")
	}
	c.id = 0
	c.status = gqlws.StatusReconnecting
	c.lastKA = time.Time{}
	for {
		err := c.connect()
		if err != nil {
			if c.reconnectBackoff.Attempt() > float64(c.ReconnectAttempts) {
				return
			}
			time.Sleep(c.reconnectBackoff.Duration())
			continue
		}
		c.reconnectBackoff.Reset()
		break
	}
}

func (c *WSClient) flushUnsentMessage() {
	for _, rawMsg := range c.unsentRawMsgQueue {
		_ = c.sendRawMessage(rawMsg)
	}
	c.unsentRawMsgQueue = nil
}

func (c *WSClient) UnderlyingConn() *websocket.Conn {
	if c == nil {
		return nil
	}
	return c.conn
}
