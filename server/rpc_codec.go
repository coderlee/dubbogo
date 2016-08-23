/******************************************************
# DESC    : provide interface for rpc_steam;
#           receive client req stream and decode them into tm(transport.message), assign tm to server.request, call Service.Method to handle request
		    and then encode server.response packet into byte stream by dubbogo.codec and send them to client by transport
# AUTHOR  : Alex Stocks
# VERSION : 1.0
# LICENCE : Apache Licence 2.0
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2016-07-20 21:37
# FILE    : rpc_codec.go
******************************************************/

package server

import (
	"bytes"
)

import (
	"github.com/AlexStocks/dubbogo/codec"
	"github.com/AlexStocks/dubbogo/codec/jsonrpc"
	"github.com/AlexStocks/dubbogo/transport"
)

type rpcCodec struct {
	socket transport.Socket
	codec  codec.Codec

	req *transport.Message
	buf *readWriteCloser
}

type readWriteCloser struct {
	wbuf *bytes.Buffer
	rbuf *bytes.Buffer
}

var (
	defaultCodecs = map[string]codec.NewCodec{
		"application/json":     jsonrpc.NewCodec,
		"application/json-rpc": jsonrpc.NewCodec,
	}
)

func (rwc *readWriteCloser) Read(p []byte) (n int, err error) {
	return rwc.rbuf.Read(p)
}

func (rwc *readWriteCloser) Write(p []byte) (n int, err error) {
	return rwc.wbuf.Write(p)
}

func (rwc *readWriteCloser) Close() error {
	rwc.rbuf.Reset()
	rwc.wbuf.Reset()
	return nil
}

func newRpcCodec(req *transport.Message, socket transport.Socket, c codec.NewCodec) serverCodec {
	rwc := &readWriteCloser{
		rbuf: bytes.NewBuffer(req.Body),
		wbuf: bytes.NewBuffer(nil),
	}
	r := &rpcCodec{
		buf:    rwc,
		codec:  c(rwc),
		req:    req,
		socket: socket,
	}
	return r
}

func (c *rpcCodec) ReadRequestHeader(r *request, first bool) error {
	m := codec.Message{Header: c.req.Header}

	if !first {
		var tm transport.Message
		// transport.Socket.Recv
		if err := c.socket.Recv(&tm); err != nil {
			return err
		}
		c.buf.rbuf.Reset()
		if _, err := c.buf.rbuf.Write(tm.Body); err != nil {
			return err
		}

		m.Header = tm.Header
	}

	err := c.codec.ReadHeader(&m, codec.Request)
	r.Service = m.Target
	r.Method = m.Method
	r.Seq = m.Id
	return err
}

func (c *rpcCodec) ReadRequestBody(b interface{}) error {
	return c.codec.ReadBody(b)
}

func (c *rpcCodec) WriteResponse(r *response, body interface{}, last bool) error {
	c.buf.wbuf.Reset()
	m := &codec.Message{
		Target: r.Service,
		Method: r.Method,
		Id:     r.Seq,
		Error:  r.Error,
		Type:   codec.Response,
		Header: map[string]string{},
	}
	if err := c.codec.Write(m, body); err != nil {
		return err
	}

	m.Header["Content-Type"] = c.req.Header["Content-Type"]
	return c.socket.Send(&transport.Message{
		Header: m.Header,
		Body:   c.buf.wbuf.Bytes(),
	})
}

func (c *rpcCodec) Close() error {
	c.buf.Close()
	c.codec.Close()
	// fmt.Println("http transport Close invoked in rpcCodec.Close")
	return c.socket.Close()
}
