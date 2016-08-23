package transport

import (
	"context"
	"time"
	// "golang.org/x/net/context"
)

type Options struct {
	Addrs []string
	// Timeout sets the timeout for Send/Recv
	Timeout time.Duration

	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

type DialOptions struct {
	Stream  bool
	Timeout time.Duration
	Path    string

	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

type ListenOptions struct {
	// Currently set in global options

	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

// Timeout sets the timeout for Send/Recv execution
func Timeout(t time.Duration) Option {
	return func(o *Options) {
		o.Timeout = t
	}
}

// Addrs to use for transport
func Addrs(addrs ...string) Option {
	return func(o *Options) {
		o.Addrs = addrs
	}
}

// Indicates whether this is a streaming connection
func WithStream() DialOption {
	return func(o *DialOptions) {
		o.Stream = true
	}
}

// Timeout used when dialling the remote side
func WithTimeout(d time.Duration) DialOption {
	return func(o *DialOptions) {
		o.Timeout = d
	}
}

func WithPath(path string) DialOption {
	return func(o *DialOptions) {
		o.Path = path
	}
}
