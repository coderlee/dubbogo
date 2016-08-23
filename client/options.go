/******************************************************
# DESC    :
# AUTHOR  : Alex Stocks
# VERSION : 1.0
# LICENCE : Apache Licence 2.0
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2016-06-30 10:44
# FILE    : options.go
******************************************************/

package client

import (
	"context"
	"time"
	// "golang.org/x/net/context"
)

import (
	"github.com/AlexStocks/dubbogo/codec"
	"github.com/AlexStocks/dubbogo/registry"
	"github.com/AlexStocks/dubbogo/selector"
	"github.com/AlexStocks/dubbogo/transport"
)

type Options struct {
	// Used to select codec
	ContentType string

	// Plugged interfaces
	Codecs    map[string]codec.NewCodec
	Registry  registry.Registry
	Selector  selector.Selector
	Transport transport.Transport

	// Connection Pool
	PoolSize int
	PoolTTL  time.Duration

	// Default Call Options
	CallOptions CallOptions

	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

type CallOptions struct {
	// Transport Dial Timeout
	DialTimeout time.Duration
	// Number of Call attempts
	Retries int
	// Request/Response timeout
	RequestTimeout time.Duration

	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

type RequestOptions struct {
	Stream bool

	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

func newOptions(options ...Option) Options {
	opts := Options{
		Codecs: make(map[string]codec.NewCodec),
		CallOptions: CallOptions{
			Retries:        DefaultRetries,
			RequestTimeout: DefaultRequestTimeout,
			DialTimeout:    transport.DefaultDialTimeout,
		},
		PoolSize: DefaultPoolSize,
		PoolTTL:  DefaultPoolTTL,
	}

	for _, o := range options {
		o(&opts)
	}

	if opts.Registry == nil {
		panic("client.Options.Registry is nil")
	}

	if opts.Selector == nil {
		panic("client.Options.Selector is nil")
	}

	if len(opts.ContentType) == 0 {
		opts.ContentType = defaultContentType
	}

	return opts
}

// Codec to be used to encode/decode requests for a given content type
func Codec(contentType string, c codec.NewCodec) Option {
	return func(o *Options) {
		o.Codecs[contentType] = c
	}
}

// Default content type of the client
func ContentType(ct string) Option {
	return func(o *Options) {
		o.ContentType = ct
	}
}

// PoolSize sets the connection pool size
func PoolSize(d int) Option {
	return func(o *Options) {
		o.PoolSize = d
	}
}

// PoolSize sets the connection pool size
func PoolTTL(d time.Duration) Option {
	return func(o *Options) {
		o.PoolTTL = d
	}
}

// Registry to find nodes for a given service
func Registry(r registry.Registry) Option {
	return func(o *Options) {
		o.Registry = r
	}
}

// Transport to use for communication e.g http, rabbitmq, etc
func Transport(t transport.Transport) Option {
	return func(o *Options) {
		o.Transport = t
	}
}

// Select is used to select a node to route a request to
func Selector(s selector.Selector) Option {
	return func(o *Options) {
		o.Selector = s
	}
}

// Number of retries when making the request.
// Should this be a Call Option?
func Retries(i int) Option {
	return func(o *Options) {
		o.CallOptions.Retries = i
	}
}

// The request timeout.
// Should this be a Call Option?
func RequestTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.CallOptions.RequestTimeout = d
	}
}

// Transport dial timeout
func DialTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.CallOptions.DialTimeout = d
	}
}

// Call Options

// WithRetries is a CallOption which overrides that which
// set in Options.CallOptions
func WithRetries(i int) CallOption {
	return func(o *CallOptions) {
		o.Retries = i
	}
}

// WithRequestTimeout is a CallOption which overrides that which
// set in Options.CallOptions
func WithRequestTimeout(d time.Duration) CallOption {
	return func(o *CallOptions) {
		o.RequestTimeout = d
	}
}

// WithDialTimeout is a CallOption which overrides that which
// set in Options.CallOptions
func WithDialTimeout(d time.Duration) CallOption {
	return func(o *CallOptions) {
		o.DialTimeout = d
	}
}

// Request Options

func StreamingRequest() RequestOption {
	return func(o *RequestOptions) {
		o.Stream = true
	}
}
