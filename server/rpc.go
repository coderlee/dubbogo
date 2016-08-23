/******************************************************
# DESC    : rpc server for dubbog provider
# AUTHOR  : Alex Stocks
# VERSION : 1.0
# LICENCE : Apache Licence 2.0
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2016-07-21 17:00
# FILE    : rpc.go
******************************************************/

package server

import (
	"context"
	"fmt"
	"runtime/debug"
	"strconv"
	"sync"
	"time"
	// "golang.org/x/net/context"
)

import (
	log "github.com/AlexStocks/log4go"
)

import (
	"github.com/AlexStocks/dubbogo/codec"
	"github.com/AlexStocks/dubbogo/common"
	"github.com/AlexStocks/dubbogo/registry"
	"github.com/AlexStocks/dubbogo/transport"
)

// 完成注册任务
type rpcServer struct {
	rpc  []*server // 处理外部请求,改为数组形式,以监听多个地址
	done chan struct{}
	once sync.Once

	sync.RWMutex
	opts     Options            // codec,transport,registry
	handlers map[string]Handler // interface -> Handler
	wg       sync.WaitGroup
}

func newRpcServer(opts ...Option) Server {
	var (
		num int
	)
	options := newOptions(opts...)
	servers := make([]*server, len(options.ServerConfList))
	num = len(options.ServerConfList)
	for i := 0; i < num; i++ {
		servers[i] = initServer()
	}
	return &rpcServer{
		opts:     options,
		rpc:      servers,
		handlers: make(map[string]Handler),
		done:     make(chan struct{}),
	}
}

func (this *rpcServer) handlePkg(servo interface{}, sock transport.Socket) {
	var (
		ok          bool
		rpc         *server
		msg         transport.Message
		err         error
		timeout     uint64
		contentType string
		codecFunc   codec.NewCodec
		codec       serverCodec
		header      map[string]string
		key         string
		value       string
		ctx         context.Context
	)

	if rpc, ok = servo.(*server); !ok {
		return
	}

	defer func() { // panic执行之前会保证defer被执行
		if r := recover(); r != nil {
			log.Warn("connection{local:%v, remote:%v} panic error:%#v, debug stack:%s",
				sock.LocalAddr(), sock.RemoteAddr(), r, string(debug.Stack()))
		}

		// close socket
		sock.Close() // 在这里保证了整个逻辑执行完毕后，关闭了连接，回收了socket fd
	}()

	for {
		msg.Reset()
		// 读取请求包
		if err = sock.Recv(&msg); err != nil {
			return
		}

		// 下面的所有逻辑都是处理请求包，并回复response
		// we use this Content-Type header to identify the codec needed
		contentType = msg.Header["Content-Type"]

		// codec of jsonrpc & other type etc
		codecFunc, err = this.newCodec(contentType)
		if err != nil {
			sock.Send(&transport.Message{
				Header: map[string]string{
					"Content-Type": "text/plain",
				},
				Body: []byte(err.Error()),
			})
			return
		}

		// !!!! 雷同于consumer/rpc_client中那个关键的一句，把github.com/AlexStocks/dubbogo/transport & github.com/AlexStocks/dubbogo/codec结合了起来
		// newRpcCodec(*transport.Message, transport.Socket, codec.NewCodec)
		codec = newRpcCodec(&msg, sock, codecFunc)

		// strip our headers
		header = make(map[string]string)
		for key, value = range msg.Header {
			header[key] = value
		}
		delete(header, "Content-Type")
		delete(header, "Timeout")

		// ctx = metadata.NewContext(context.Background(), header)
		ctx = context.WithValue(context.Background(), common.DUBBOGO_CTX_KEY, header)
		// we use this Timeout header to set a server deadline
		if len(msg.Header["Timeout"]) > 0 {
			if timeout, err = strconv.ParseUint(msg.Header["Timeout"], 10, 64); err == nil {
				ctx, _ = context.WithTimeout(ctx, time.Duration(timeout))
			}
		}

		if err = rpc.serveRequest(ctx, codec, contentType); err != nil {
			log.Info("Unexpected error serving request, closing socket: %v", err)
			return
		}
	}
}

func (this *rpcServer) newCodec(contentType string) (codec.NewCodec, error) {
	var (
		ok bool
		cf codec.NewCodec
	)
	if cf, ok = this.opts.Codecs[contentType]; ok {
		return cf, nil
	}
	if cf, ok = defaultCodecs[contentType]; ok {
		return cf, nil
	}
	return nil, fmt.Errorf("Unsupported Content-Type: %s", contentType)
}

func (this *rpcServer) Options() Options {
	var (
		opts Options
	)

	this.RLock()
	opts = this.opts
	this.RUnlock()

	return opts
}

/*
type ProviderServiceConfig struct {
	Protocol string // from ServiceConfig, get field{Path} from ServerConfig by this field
	Service string  // from handler, get field{Protocol, Group, Version} from ServiceConfig by this field
	Group   string
	Version string
	Methods string
	Path    string
}

type ServiceConfig struct {
	Protocol string `required:"true",default:"dubbo"` // codec string, jsonrpc etc
	Service string `required:"true"`
	Group   string
	Version string
}

type ServerConfig struct {
	Protocol string `required:"true",default:"dubbo"` // codec string, jsonrpc etc
	IP       string
	Port     int
}
*/

func (this *rpcServer) Handle(h Handler) error {
	var (
		i           int
		j           int
		serviceNum  int
		serverNum   int
		err         error
		config      Options
		serviceConf registry.ProviderServiceConfig
	)
	config = this.Options()

	serviceConf.Service = h.Service()
	serviceConf.Version = h.Version()

	serviceNum = len(config.ServiceConfList)
	serverNum = len(config.ServerConfList)
	for i = 0; i < serviceNum; i++ {
		if config.ServiceConfList[i].Service == serviceConf.Service {
			serviceConf.Protocol = config.ServiceConfList[i].Protocol
			serviceConf.Group = config.ServiceConfList[i].Group
			serviceConf.Version = config.ServiceConfList[i].Version
			for j = 0; j < serverNum; j++ {
				if config.ServerConfList[j].Protocol == serviceConf.Protocol {
					this.Lock()
					serviceConf.Methods, err = this.rpc[j].register(h)
					this.Unlock()
					if err != nil {
						return err
					}

					serviceConf.Path = config.ServerConfList[j].Address()
					err = config.Registry.Register(serviceConf)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	this.Lock()
	this.handlers[h.Service()] = h
	this.Unlock()

	return nil
}

func (this *rpcServer) Start() error {
	var (
		i         int
		serverNum int
		err       error
		config    Options
		rpc       *server
		listener  transport.Listener
	)
	config = this.Options()

	serverNum = len(config.ServerConfList)
	for i = 0; i < serverNum; i++ {
		listener, err = config.Transport.Listen(config.ServerConfList[i].Address())
		if err != nil {
			return err
		}
		log.Info("Listening on %s", listener.Addr())

		this.Lock()
		rpc = this.rpc[i]
		rpc.listener = listener
		this.Unlock()

		this.wg.Add(1)
		go func(servo *server) {
			listener.Accept(func(s transport.Socket) { this.handlePkg(rpc, s) })
			this.wg.Done()
		}(rpc)

		this.wg.Add(1)
		go func(servo *server) { // server done goroutine
			var err error
			<-this.done                  // step1: block to wait for done channel(wait rpcServer.Stop step2)
			err = servo.listener.Close() // step2: and then close listener
			if err != nil {
				log.Warn("listener{addr:%s}.Close() = error{%#v}", servo.listener.Addr(), err)
			}
			this.wg.Done()
		}(rpc)
	}

	return nil
}

func (this *rpcServer) Stop() {
	this.once.Do(func() {
		close(this.done)
		this.wg.Wait()
		if this.opts.Registry != nil {
			this.opts.Registry.Close()
			this.opts.Registry = nil
		}
	})
}

func (this *rpcServer) String() string {
	return "dubbogo rpc server"
}
