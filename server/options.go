/******************************************************
# DESC    : rpc server parameters
# AUTHOR  : Alex Stocks
# VERSION : 1.0
# LICENCE : Apache Licence 2.0
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2016-07-20 21:39
# FILE    : options.go
******************************************************/

package server

import (
	// "golang.org/x/net/context"
	"context"
)

import (
	"github.com/AlexStocks/dubbogo/codec"
	"github.com/AlexStocks/dubbogo/common"
	"github.com/AlexStocks/dubbogo/registry"
	"github.com/AlexStocks/dubbogo/transport"
)

type Options struct {
	Codecs    map[string]codec.NewCodec
	Registry  registry.Registry
	Transport transport.Transport

	ServerConfList  []registry.ServerConfig
	ServiceConfList []registry.ServiceConfig
	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

func newOptions(opt ...Option) Options {
	opts := Options{
		Codecs: make(map[string]codec.NewCodec),
	}

	for _, o := range opt {
		o(&opts)
	}

	if opts.Registry == nil {
		panic("server.Options.Registry is nil")
	}

	if opts.Transport == nil {
		panic("server.Options.Transport is nil")
	}

	return opts
}

// Codec to use to encode/decode requests for a given content type
func Codec(codecs map[string]codec.NewCodec) Option {
	return func(o *Options) {
		o.Codecs = codecs
	}
}

// Registry used for discovery
func Registry(r registry.Registry) Option {
	return func(o *Options) {
		o.Registry = r
	}
}

// Transport mechanism for communication e.g http, rabbitmq, etc
func Transport(t transport.Transport) Option {
	return func(o *Options) {
		o.Transport = t
	}
}

func ServerConfList(confList []registry.ServerConfig) Option {
	return func(o *Options) {
		o.ServerConfList = confList
		for i := 0; i < len(o.ServerConfList); i++ {
			o.ServerConfList[i].IP, _ = common.GetLocalIP(o.ServerConfList[i].IP)
		}
	}
}

func ServiceConfList(confList []registry.ServiceConfig) Option {
	return func(o *Options) {
		o.ServiceConfList = confList
	}
}
