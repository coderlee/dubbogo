package transport

import (
	"net"
	"time"
)

type Message struct {
	Header map[string]string
	Body   []byte
}

func (this *Message) Reset() {
	var key string
	this.Body = this.Body[:0]
	for key = range this.Header {
		delete(this.Header, key)
	}
}

type Socket interface {
	Recv(*Message) error
	Send(*Message) error
	Reset(c net.Conn, release func())
	Close() error
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
}

type Client interface {
	Recv(*Message) error
	Send(*Message) error
	Close() error
}

type Listener interface {
	Addr() string
	Close() error
	Accept(func(Socket)) error
}

// Transport is an interface which is used for communication between
// services. It uses socket send/recv semantics and had various
// implementations {HTTP, RabbitMQ, NATS, ...}
type Transport interface {
	Dial(addr string, opts ...DialOption) (Client, error)
	Listen(addr string, opts ...ListenOption) (Listener, error)
	String() string
}

type Option func(*Options)

type DialOption func(*DialOptions)

type ListenOption func(*ListenOptions)

var (
	DefaultDialTimeout = time.Second * 5
)
