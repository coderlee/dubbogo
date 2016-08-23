/******************************************************
# DESC    : apply client interface for app
# AUTHOR  : Alex Stocks
# VERSION : 1.0
# LICENCE : Apache Licence 2.0
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2016-06-28 16:43
# FILE    : rpc_client.go
******************************************************/

package client

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
	// "golang.org/x/net/context"
)

import (
	log "github.com/AlexStocks/log4go"
)

import (
	"github.com/AlexStocks/dubbogo/codec"
	"github.com/AlexStocks/dubbogo/common"
	"github.com/AlexStocks/dubbogo/selector"
	"github.com/AlexStocks/dubbogo/transport"
)

const (
	CLEAN_CHANNEL_SIZE = 64
)

// thread safe
type rpcClient struct {
	ID   uint64
	once sync.Once
	opts Options
	pool *pool

	// gc goroutine
	done chan struct{}
	wg   sync.WaitGroup
	gcCh chan interface{}
}

func newRPCClient(opt ...Option) Client {
	opts := newOptions(opt...)

	t := time.Now()
	rc := &rpcClient{
		ID:   uint64(t.Second() * t.Nanosecond() * common.Goid()),
		once: sync.Once{},
		opts: opts,
		pool: newPool(opts.PoolSize, opts.PoolTTL),
		done: make(chan struct{}),
		gcCh: make(chan interface{}, CLEAN_CHANNEL_SIZE),
	}
	log.Info("client initial ID:%d", rc.ID)
	rc.wg.Add(1)
	go rc.gc()

	c := Client(rc)

	return c
}

// rpcClient garbage collector
func (this *rpcClient) gc() {
	var (
		obj interface{}
	)

	defer this.wg.Done()
LOOP:
	for {
		select {
		case <-this.done:
			log.Info("(rpcClient)gc goroutine exit now ...")
			break LOOP
		case obj = <-this.gcCh:
			switch obj.(type) {
			case *rpcStream:
				obj.(*rpcStream).Close() // stream.Close()->rpcPlusCodec.Close->poolConn.Close->httpTransportClient.Close
			}
		}
	}
}

func (this *rpcClient) newCodec(contentType string) (codec.NewCodec, error) {
	if c, ok := this.opts.Codecs[contentType]; ok {
		return c, nil
	}

	if cf, ok := defaultCodecs[contentType]; ok {
		return cf, nil
	}

	return nil, fmt.Errorf("Unsupported Content-Type: %s", contentType)
}

// 流程
// 1 创建transport.Message对象 msg;
// 2 设置msg.Header;
// 3 创建codec对象;
// 4 从连接池中获取一个连接conn;
// 5 创建stream对象;
// 6 启动一个收发goroutine, 调用stream完成网络收发;
// 7 通过一个error channel等待收发goroutine结束流程。
// rpc client -> rpc stream -> rpc codec -> codec + transport
func (this *rpcClient) call(reqID uint64, ctx context.Context, address string, path string, req Request, rsp interface{}, opts CallOptions) error {
	msg := &transport.Message{
		Header: make(map[string]string),
	}

	md, ok := ctx.Value(common.DUBBOGO_CTX_KEY).(map[string]string)
	if ok {
		for k, v := range md {
			msg.Header[k] = v
		}
	}

	// set timeout in nanoseconds
	msg.Header["Timeout"] = fmt.Sprintf("%d", opts.RequestTimeout)
	// set the content type for the request
	msg.Header["Content-Type"] = req.ContentType()
	msg.Header["Accept"] = req.ContentType()

	// 创建codec
	cf, err := this.newCodec(req.ContentType())
	if err != nil {
		return common.InternalServerError("dubbogo.client", err.Error())
	}

	// 从连接池获取连接对象
	var grr error
	c, err := this.pool.getConn(address, this.opts.Transport, transport.WithTimeout(opts.DialTimeout), transport.WithPath(path))
	if err != nil {
		return common.InternalServerError("dubbogo.client", fmt.Sprintf("Error sending request: %v", err))
	}

	// 网络层请求
	stream := &rpcStream{
		seq:     reqID,
		context: ctx,
		request: req,
		closed:  make(chan bool),
		// !!!!! 这个codec是rpc_codec,其主要成员是发送内容msg，网络层(transport)对象c，codec对象cf
		// 这行代码把github.com/AlexStocks/dubbogo/codec dubbo/client github.com/AlexStocks/dubbogo/transport连接了起来
		// newRpcPlusCodec(*transport.Message, transport.Client, codec.Codec)
		codec: newRpcPlusCodec(msg, c, cf),
	}
	defer func() {
		log.Debug("check request{%#v}, stream condition before store the conn object into pool", req)
		// defer execution of release
		if req.Stream() {
			// 只缓存长连接
			this.pool.release(address, c, grr)
		}
		// 下面这个分支与(rpcStream)Close, 2016/08/07
		// else {
		// 	log.Debug("close pool connection{%#v}", c)
		// 	c.Close() // poolConn.Close->httpTransportClient.Close
		// }
		this.gcCh <- stream
	}()

	ch := make(chan error, 1)

	go func() {
		var (
			err error
		)
		defer func() {
			if panicMsg := recover(); panicMsg != nil {
				if msg, ok := panicMsg.(string); ok {
					ch <- common.InternalServerError("dubbogo.client", strconv.Itoa(int(stream.seq))+" request error, panic msg:"+msg)
				} else {
					ch <- common.InternalServerError("dubbogo.client", "request error")
				}
			}
		}()

		// send request
		// 1 stream的send函数调用rpcStream.clientCodec.WriteRequest函数(从line 119可见clientCodec实际是newRpcPlusCodec);
		// 2 rpcPlusCodec.WriteRequest调用了codec.Write(codec.Message, body)，在给request赋值后，然后又调用了transport.Send函数
		// 3 httpTransportClient根据m{header, body}拼凑http.Request{header, body}，然后再调用http.Request.Write把请求以tcp协议的形式发送出去
		if err = stream.Send(req.Request()); err != nil {
			ch <- err
			return
		}

		// recv request
		// 1 stream.Recv 调用rpcPlusCodec.ReadResponseHeader & rpcPlusCodec.ReadResponseBody;
		// 2 rpcPlusCodec.ReadResponseHeader 先调用httpTransportClient.read，然后再调用codec.ReadHeader
		// 3 rpcPlusCodec.ReadResponseBody 调用codec.ReadBody
		if err = stream.Recv(rsp); err != nil {
			log.Warn("stream.Recv(ID{%d}, req{%#v}, rsp{%#v}) = err{%t}", reqID, req, rsp, err)
			ch <- err
			return
		}

		// success
		ch <- nil
	}()

	select {
	case err := <-ch:
		grr = err
		return err
	case <-ctx.Done():
		grr = ctx.Err()
		return common.New("dubbogo.client", fmt.Sprintf("%v", ctx.Err()), 408)
	}
}

func (this *rpcClient) Init(opts ...Option) error {
	size := this.opts.PoolSize
	ttl := this.opts.PoolTTL

	for _, o := range opts {
		o(&this.opts)
	}

	// recreate the pool if the options changed
	if size != this.opts.PoolSize || ttl != this.opts.PoolTTL {
		this.pool = newPool(this.opts.PoolSize, this.opts.PoolTTL)
	}

	return nil
}

func (this *rpcClient) Options() Options {
	return this.opts
}

// 流程
// 1 从selector中根据service选择一个provider，具体的来说，就是next函数对象;
// 2 构造call函数;
//   2.1 调用next函数返回provider的serviceurl;
//   2.2 调用rpcClient.call()
// 3 根据重试次数的设定，循环调用call，直到有一次成功或者重试
func (this *rpcClient) Call(ctx context.Context, request Request, response interface{}, opts ...CallOption) error {
	reqID := atomic.AddUint64(&this.ID, 1)
	// make a copy of call opts
	callOpts := this.opts.CallOptions
	for _, opt := range opts {
		opt(&callOpts)
	}

	// get next nodes from the selector
	next, err := this.opts.Selector.Select(request.Service())
	if err != nil && err == selector.ErrNotFound {
		log.Error("selector.Select(request{%#v}) = error{%#v}", request, err)
		return common.NotFound("dubbogo.client", err.Error())
	} else if err != nil {
		log.Error("selector.Select(request{%#v}) = error{%#v}", request, err)
		return common.InternalServerError("dubbogo.client", err.Error())
	}

	// check if we already have a deadline
	d, ok := ctx.Deadline()
	if !ok {
		// no deadline so we create a new one
		ctx, _ = context.WithTimeout(ctx, callOpts.RequestTimeout)
	} else {
		// got a deadline so no need to setup context
		// but we need to set the timeout we pass along
		opt := WithRequestTimeout(d.Sub(time.Now()))
		opt(&callOpts)
	}

	// should we noop right here?
	select {
	case <-ctx.Done():
		return common.New("dubbogo.client", fmt.Sprintf("%v", ctx.Err()), 408)
	default:
	}

	// return errors.New("dubbogo.client", "request timeout", 408)
	call := func(i int) error {
		// select next node
		serviceURL, err := next(reqID)
		if err != nil && err == selector.ErrNotFound {
			log.Error("selector.next(request{%#v}, reqID{%d}) = error{%#v}", request, reqID, err)
			return common.NotFound("dubbogo.client", err.Error())
		} else if err != nil {
			log.Error("selector.next(request{%#v}, reqID{%d}) = error{%#v}", request, reqID, err)
			return common.InternalServerError("dubbogo.client", err.Error())
		}

		// set the address
		address := serviceURL.Location //  + serviceURL.Path
		// make the call
		err = this.call(reqID, ctx, address, serviceURL.Path, request, response, callOpts)
		log.Debug("@i{%d}, call(ID{%v}, ctx{%v}, address{%v}, path{%v}, request{%v}, response{%v}) = err{%v}",
			i, reqID, ctx, address, serviceURL.Path, request, response, err)
		this.opts.Selector.Mark(request.Service(), serviceURL, err)
		return err
	}

	var (
		gerr error
		ch   chan error
	)
	ch = make(chan error, callOpts.Retries)
	for i := 0; i < callOpts.Retries; i++ {
		go func(index int) {
			ch <- call(index)
		}(i)

		select {
		case <-ctx.Done():
			log.Error("reqID{%d}, @i{%d}, ctx.Done()", reqID, i)
			return common.New("dubbogo.client", fmt.Sprintf("%v", ctx.Err()), 408)
		case err := <-ch:
			// if the call succeeded lets bail early
			if err == nil || len(err.Error()) == 0 {
				return nil
			}
			log.Error("reqID{%d}, @i{%d}, err{%T-%v}", reqID, i, err, err)
			gerr = err
		}
	}

	return gerr
}

func (this *rpcClient) NewJsonRequest(service string, method string, request interface{}, reqOpts ...RequestOption) Request {
	return newRpcRequest(service, method, request, "application/json", reqOpts...)
}

func (this *rpcClient) String() string {
	return "dubbogo rpc client"
}

func (this *rpcClient) Close() {
	close(this.done)
	this.wg.Wait()
	this.once.Do(func() {
		if this.opts.Selector != nil {
			this.opts.Selector.Close()
			this.opts.Selector = nil
		}
		if this.opts.Registry != nil {
			this.opts.Registry.Close()
			this.opts.Registry = nil
		}
	})
}
