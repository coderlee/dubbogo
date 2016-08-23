/******************************************************
# DESC    : invoke dubbogo.codec & dubbogo.transport handle client package/streaming request
# AUTHOR  : Alex Stocks
# VERSION : 1.0
# LICENCE : Apache Licence 2.0
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2016-07-20 21:46
# FILE    : rpc_stream.go
******************************************************/

package server

import (
	"context"
	"log"
	"sync"
	// "golang.org/x/net/context"
)

// Implements the Streamer interface
type rpcStream struct {
	sync.RWMutex
	seq     uint64
	closed  bool
	err     error
	request Request
	codec   serverCodec
	context context.Context
}

func (r *rpcStream) Context() context.Context {
	return r.context
}

func (r *rpcStream) Request() Request {
	return r.request
}

func (r *rpcStream) Send(msg interface{}) error {
	r.Lock()
	// defer r.Unlock()

	resp := response{
		// ServiceMethod: r.request.Method(),
		Service: r.request.Service(),
		Method:  r.request.Method(),
		Seq:     r.seq,
	}

	err := r.codec.WriteResponse(&resp, msg, false)
	r.Unlock()
	if err != nil {
		log.Println("rpc: writing response:", err)
	}

	return err
}

func (r *rpcStream) Recv(msg interface{}) error {
	r.Lock()
	defer r.Unlock()

	req := request{}

	if err := r.codec.ReadRequestHeader(&req, false); err != nil {
		// discard body
		r.codec.ReadRequestBody(nil)
		return err
	}

	// we need to stay up to date with sequence numbers
	r.seq = req.Seq

	if err := r.codec.ReadRequestBody(msg); err != nil {
		return err
	}

	return nil
}

func (r *rpcStream) Error() error {
	r.RLock()
	defer r.RUnlock()
	return r.err
}

func (r *rpcStream) Close() error {
	r.Lock()
	defer r.Unlock()
	r.closed = true
	// fmt.Println("rpcStream.Close")
	// rpcCodec.Close -> rpcCodec.socket.Close -> transport.Close()，
	// 但其实这个路线不会被执行，参见func (this *rpcServer) handlePkg(servo interface{}, sock transport.Socket)里面的defer语句块，
	// 只有transport.Close()会被执行
	return r.codec.Close()
}
