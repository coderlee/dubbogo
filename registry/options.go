package registry

import (
	// "golang.org/x/net/context"
	"context"
)

import (
	"github.com/AlexStocks/dubbogo/common"
)

type Options struct {
	common.ApplicationConfig
	RegistryConfig

	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

// Option used to initialise the client
type Option func(*Options)

func ApplicationConf(conf common.ApplicationConfig) Option {
	return func(o *Options) {
		o.ApplicationConfig = conf
	}
}

func RegistryConf(conf RegistryConfig) Option {
	return func(o *Options) {
		o.RegistryConfig = conf
	}
}

func Context(ctx context.Context) Option {
	return func(o *Options) {
		o.Context = ctx
	}
}
