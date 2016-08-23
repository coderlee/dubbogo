/******************************************************
# DESC    : code flow of handlePag
# AUTHOR  : Alex Stocks
# VERSION : 1.0
# LICENCE : Apache Licence 2.0
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2016-07-21 15:59
# FILE    : doc.go
******************************************************/

package server

/*
// consumer: rpc client -> rpc stream -> rpc codec -> transport + codec
// provider: rpc server -> rpc stream -> rpc codec -> transport + codec
func (this *rpcServer) handlePkg(servo interface{}, sock transport.Socket) {
	sock.Recv(&msg) // msg = transport.Message
	// func (r *rpcStream) Recv(msg interface{}) error {
	// 	 r.codec.ReadRequestHeader(&req, false)
	// 	 // func (c *rpcCodec) ReadRequestHeader(r *request, first bool) error
	// 	 //   c.socket.Recv(&tm) // tm(transport.Message)
	// 	 //   //  func (h *httpTransportSocket) Recv(m *Message) error { // 读取全部reqeust，并赋值给m(transport.Message)
	// 	 //   //    http.ReadRequest(h.buff)
	// 	 //   //    ioutil.ReadAll(r.Body)
	// 	 //   //    m.Target = m.Header["Path"]
	// 	 //   //  }
	// 	 //
	// 	 //   err := c.codec.ReadHeader(&m, codec.Request)
	//   //   // func (j *jsonCodec) ReadHeader(m *codec.Message, mt codec.MessageType)
	//   //   //   case codec.Request:
	//   //   //   return j.s.ReadHeader(m)
	// 	 //   //   // func (c *serverCodec) ReadHeader(m *codec.Message) error { // serverCodec, github.com/AlexStocks/dubbogo/codec
	// 	 //   //   //   c.dec.Decode(&raw)
	// 	 //   //   //   json.Unmarshal(raw, &c.req) // 注意此处，c.req存储了请求的body
	// 	 //   //   //   m.Id = c.seq
	// 	 //   //   //   m.Method = c.req.Method
	// 	 //	  //   //   m.Target = m.Header["Path"]
	// 	 //   //   // }
	//   //   // }
	// 	 //   r.Service = m.Target
	// 	 //   r.Method = m.Method
	// 	 //   r.Seq = m.Id
	// 	 //   return err
	// 	 // } //  (c *rpcCodec) ReadRequestHeader
	// 	 r.codec.ReadRequestBody(msg)
	// }

	codecFunc, err = this.newCodec(contentType) // dubbogo.codec
	codec = newRpcCodec(&msg, sock, codecFunc)
	rpc.serveRequest(ctx, codec, contentType)
	// func (server *server) serveRequest(ctx context.Context, codec serverCodec, ct string) error {
	//   server.readRequest(codec)
	//   // func (server *server) readRequest(codec serverCodec) {
	//   //   server.readRequestHeader(codec)
	//   //   // func (server *server) readRequestHeader(codec serverCodec)
	//   //   //   err = codec.ReadRequestHeader(req, true) // 注意此时first为false，避免进行网络收发，只读取相关分析结果
	//   //   //   // func (c *rpcCodec) ReadRequestHeader(r *request, first bool) error {
	//   //   //   //   m := codec.Message{Header: c.req.Header}
	//   //   //   //   err := c.codec.ReadHeader(&m, codec.Request)
	//   //   //   //   r.Service = m.Target
	//   //   //   //   r.Method = m.Method
	//   //   //   //   r.Seq = m.Id
	//   //   //   //   return err
	//   //   //   //   }
	//   //   //   service = server.serviceMap[req.Service] // 根据Service
	//   //   //   mtype = service.method[req.Method] // 获取method, 供下面的call调用
	//   //   // }
	//   //   codec.ReadRequestBody(argv.Interface()) // rpcCodec.ReadRequestBody
	//   //   // func (c *rpcCodec) ReadRequestBody(b interface{}) error {
	//   //   //   return c.codec.ReadBody(b)
	//   //   //   //  func (c *serverCodec) ReadBody(x interface{}) error {
	//   //   //   //    json.Unmarshal(*c.req.Params, x) // decode request body, c.req value line 19
	//   //   //   //  }
	//   //   // }
	//   // }
	//   service.call()
	// }
}
*/
