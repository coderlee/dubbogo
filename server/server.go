/******************************************************
# DESC    : provider interfaces
# AUTHOR  : Alex Stocks
# VERSION : 1.0
# LICENCE : Apache Licence 2.0
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2016-07-20 21:48
# FILE    : provider.go
******************************************************/

package server

import (
	"context"
	// "golang.org/x/net/context"
)

// Handler interface represents a Service request handler. It's generated
// by passing any type of public concrete object with methods into server.NewHandler.
// Most will pass in a struct.
//
// Example:
//
//	type Hello struct {}
//
//	func (s *Hello) Method(context, request, response) error {
//		return nil
//	}
//
//  func (s *Hello) Service() string {
//      return "com.youni.service"
//  }
//
//  func (s *Hello) Version() string {
//      return "1.0.0"
//  }

type Handler interface {
	Service() string // Service Interface
	Version() string
}

// Provider
type Server interface {
	Options() Options
	Handle(Handler) error
	Start() error
	Stop()
	String() string
}

type Request interface {
	Service() string
	Method() string
	ContentType() string
	Request() interface{}
	// indicates whether the request will be streamed
	Stream() bool
}

// Streamer represents a stream established with a client.
// A stream can be bidirectional which is indicated by the request.
// The last error will be left in Error().
// EOF indicated end of the stream.
type Streamer interface {
	Context() context.Context
	Request() Request
	Send(interface{}) error
	Recv(interface{}) error
	Error() error
	Close() error
}

type Option func(*Options)

func NewServer(opts ...Option) Server {
	return newRpcServer(opts...)
}
