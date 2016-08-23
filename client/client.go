/******************************************************
# DESC    :
# AUTHOR  : Alex Stocks
# VERSION : 1.0
# LICENCE : Apache Licence 2.0
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2016-06-30 10:45
# FILE    : client.go
******************************************************/

package client

import (
	"context"
	"time"
	// "golang.org/x/net/context"
)

// Client is the interface used to make requests to services.
// It supports Request/Response via Transport and Publishing via the Broker.
// It also supports bidiectional streaming of requests.
type Client interface {
	Init(...Option) error
	Options() Options
	NewJsonRequest(service, method string, req interface{}, reqOpts ...RequestOption) Request
	Call(ctx context.Context, req Request, rsp interface{}, opts ...CallOption) error
	String() string
	Close()
}

// Request is the interface for a synchronous request used by Call or Stream
type Request interface {
	Service() string
	Method() string
	ContentType() string
	Request() interface{}
	// indicates whether the request will be a streaming one rather than unary
	Stream() bool
}

// Streamer is the inteface for a bidirectional synchronous stream
type Streamer interface {
	Context() context.Context
	Request() Request
	Send(interface{}) error
	Recv(interface{}) error
	Error() error
	Close() error
}

// Option used by the Client
type Option func(*Options)

// CallOption used by Call or Stream
type CallOption func(*CallOptions)

// RequestOption used by NewRequest
type RequestOption func(*RequestOptions)

var (
	// DefaultRetries is the default number of times a request is tried
	DefaultRetries = 1
	// DefaultRequestTimeout is the default request timeout
	DefaultRequestTimeout = time.Second * 5
	// DefaultPoolSize sets the connection pool size
	DefaultPoolSize = 0
	// DefaultPoolTTL sets the connection pool ttl
	DefaultPoolTTL = time.Minute
)

// creates a new client with the options passed in
func NewClient(opt ...Option) Client {
	return newRPCClient(opt...)
}
