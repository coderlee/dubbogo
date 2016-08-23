package transport

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

import (
	log "github.com/AlexStocks/log4go"
)

const (
	PathPrefix                byte = byte('/')
	DefaultMaxSleepTime            = 1 * time.Second  // accept中间最大sleep interval
	DefaultMAXConnNum              = 50 * 1024 * 1024 // 默认最大连接数 50w
	DefaultHTTPRspBufferSize       = 1024
	DefaultTcpReadBufferSize       = 128 * 1024 // 64k
	DefaultTcpWriteBufferSize      = 512 * 1024 // 64k
)

type buffer struct {
	io.ReadWriter
}

func (b *buffer) Close() error {
	return nil
}

//////////////////////////////////////////////
// http transport client
//////////////////////////////////////////////

type httpTransportClient struct {
	ht       *httpTransport
	addr     string
	conn     net.Conn
	dialOpts DialOptions
	once     sync.Once

	sync.Mutex
	r    chan *http.Request
	bl   []*http.Request
	buff *bufio.Reader
}

func initHttpTransportClient(
	ht *httpTransport,
	addr string,
	conn net.Conn,
	opts DialOptions,
) *httpTransportClient {

	return &httpTransportClient{
		ht:       ht,
		addr:     addr,
		conn:     conn,
		buff:     bufio.NewReader(conn),
		dialOpts: opts,
		r:        make(chan *http.Request, 1),
	}
}

func (h *httpTransportClient) Send(m *Message) error {
	header := make(http.Header)

	// http.header = m.header
	for k, v := range m.Header {
		header.Set(k, v)
	}

	// http.body = m.body
	reqB := bytes.NewBuffer(m.Body)
	defer reqB.Reset()
	buf := &buffer{
		reqB,
	}

	req := &http.Request{
		Method: "POST",
		URL: &url.URL{
			Scheme: "http",
			Host:   h.addr,
			Path:   h.dialOpts.Path,
		},
		Header:        header, // m.header
		Body:          buf,    // m.body
		ContentLength: int64(reqB.Len()),
		Host:          h.addr,
	}

	h.Lock()
	h.bl = append(h.bl, req)
	select {
	case h.r <- h.bl[0]:
		h.bl = h.bl[1:]
	default:
	}
	h.Unlock()

	// set timeout if its greater than 0
	if h.ht.opts.Timeout > time.Duration(0) {
		h.conn.SetWriteDeadline(time.Now().Add(h.ht.opts.Timeout))
	}

	err := req.Write(h.conn)

	if h.ht.opts.Timeout > time.Duration(0) {
		h.conn.SetWriteDeadline(time.Time{}) // learn from samuel/go-zookeeper
	}

	return err
}

func (h *httpTransportClient) Recv(m *Message) error {
	var r *http.Request
	if !h.dialOpts.Stream {
		rc, ok := <-h.r
		if !ok {
			return io.EOF
		}
		r = rc
	}

	h.Lock()
	defer h.Unlock()
	if h.buff == nil {
		return io.EOF
	}

	// set timeout if its greater than 0
	if h.ht.opts.Timeout > time.Duration(0) {
		h.conn.SetReadDeadline(time.Now().Add(h.ht.opts.Timeout))
	}

	rsp, err := http.ReadResponse(h.buff, r)
	if err != nil {
		if h.ht.opts.Timeout > time.Duration(0) {
			h.conn.SetReadDeadline(time.Time{}) // refer to samuel/go-zookeeper
		}
		return err
	}
	defer rsp.Body.Close() // 这句话如果不调用，连接在调用(httpTransportClient)Close之前就不释放

	b, err := ioutil.ReadAll(rsp.Body)
	if h.ht.opts.Timeout > time.Duration(0) {
		h.conn.SetReadDeadline(time.Time{}) // refer to samuel/go-zookeeper
	}
	if err != nil {
		return err
	}

	if rsp.StatusCode != 200 {
		return errors.New(rsp.Status + ": " + string(b))
	}

	mr := &Message{
		Header: make(map[string]string),
		Body:   b,
	}

	for k, v := range rsp.Header {
		if len(v) > 0 {
			mr.Header[k] = v[0]
		} else {
			mr.Header[k] = ""
		}
	}

	*m = *mr
	return nil
}

func (h *httpTransportClient) Close() error {
	// log.Debug("close transport client{%#v}", h)
	var err error
	h.once.Do(func() {
		h.Lock()
		h.buff.Reset(nil)
		h.buff = nil
		h.Unlock()
		close(h.r)
		err = h.conn.Close()
	})
	return err
}

//////////////////////////////////////////////
// http transport socket
//////////////////////////////////////////////

// 从下面代码来看，socket是为下面的listener服务的
type httpTransportSocket struct {
	ht      *httpTransport
	reqQ    chan *http.Request
	conn    net.Conn
	once    *sync.Once
	release func()
	sync.Mutex
	bufReader *bufio.Reader
	rspBuf    *bytes.Buffer
}

const (
	REQ_Q_SIZE = 1 // http1.1形式的短连接，一次也只能处理一个请求，放大size无意义
)

func initHttpTransportSocket(ht *httpTransport, c net.Conn, release func()) *httpTransportSocket {
	return &httpTransportSocket{
		ht:        ht,
		conn:      c,
		once:      &sync.Once{},
		bufReader: bufio.NewReader(c),
		rspBuf:    bytes.NewBuffer(make([]byte, DefaultHTTPRspBufferSize)),
		reqQ:      make(chan *http.Request, REQ_Q_SIZE),
		release:   release,
	}
}

func (h *httpTransportSocket) Reset(c net.Conn, release func()) {
	h.Close()
	h.conn = c
	h.once = &sync.Once{}
	h.release = release
}

func (h *httpTransportSocket) Recv(m *Message) error {
	if m == nil {
		return errors.New("message passed in is nil")
	}

	// set timeout if its greater than 0
	if h.ht.opts.Timeout > time.Duration(0) {
		h.conn.SetReadDeadline(time.Now().Add(h.ht.opts.Timeout))
	}

	r, err := http.ReadRequest(h.bufReader)
	if err != nil {
		if h.ht.opts.Timeout > time.Duration(0) {
			h.conn.SetReadDeadline(time.Time{}) // refer to samuel/go-zookeeper
		}
		return err
	}

	b, err := ioutil.ReadAll(r.Body)
	if h.ht.opts.Timeout > time.Duration(0) {
		h.conn.SetReadDeadline(time.Time{}) // refer to samuel/go-zookeeper
	}
	if err != nil {
		return err
	}
	r.Body.Close()

	// 初始化的时候创建了Header，并给Body赋值
	mr := &Message{
		Header: make(map[string]string),
		Body:   b,
	}

	// 下面的代码块给Message{Header}进行赋值
	for k, v := range r.Header {
		if len(v) > 0 {
			mr.Header[k] = v[0]
		} else {
			mr.Header[k] = ""
		}
	}
	mr.Header["Path"] = r.URL.Path[1:] // to get service name
	if r.URL.Path[0] != PathPrefix {
		mr.Header["Path"] = r.URL.Path
	}
	mr.Header["HttpMethod"] = r.Method

	select {
	case h.reqQ <- r:
	default:
	}

	*m = *mr
	return nil
}

func (h *httpTransportSocket) Send(m *Message) error {
	b := bytes.NewBuffer(m.Body)
	defer b.Reset()

	r := <-h.reqQ

	rsp := &http.Response{
		Header:        r.Header,   // Header先复用request的Header
		Body:          &buffer{b}, // Body
		Status:        "200 OK",
		StatusCode:    200,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		ContentLength: int64(len(m.Body)),
	}

	// 根据@m，修改Response{Header}
	for k, v := range m.Header {
		rsp.Header.Set(k, v)
	}

	select {
	case h.reqQ <- r:
	default:
	}

	// return rsp.Write(h.conn)
	h.rspBuf.Reset()
	err := rsp.Write(h.rspBuf)
	if err != nil {
		return err
	}

	// set timeout if its greater than 0
	if h.ht.opts.Timeout > time.Duration(0) {
		h.conn.SetWriteDeadline(time.Now().Add(h.ht.opts.Timeout))
	}

	_, err = h.rspBuf.WriteTo(h.conn)

	// set timeout if its greater than 0
	if h.ht.opts.Timeout > time.Duration(0) {
		h.conn.SetWriteDeadline(time.Time{}) // refer to samuel/go-zookeeper
	}

	return err
}

func (h *httpTransportSocket) error(m *Message) error {
	b := bytes.NewBuffer(m.Body)
	defer b.Reset()
	rsp := &http.Response{
		Header:        make(http.Header),
		Body:          &buffer{b},
		Status:        "500 Internal Server Error",
		StatusCode:    500,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		ContentLength: int64(len(m.Body)),
	}

	for k, v := range m.Header {
		rsp.Header.Set(k, v)
	}

	// return rsp.Write(h.conn)
	h.rspBuf.Reset()
	err := rsp.Write(h.rspBuf)
	if err != nil {
		return err
	}

	_, err = h.rspBuf.WriteTo(h.conn)
	return err
}

func (h *httpTransportSocket) Close() error {
	log.Debug("httpTransportSocket.Close")
	var err error
	h.once.Do(func() {
		h.Lock()
		h.bufReader.Reset(nil)
		h.bufReader = nil
		h.rspBuf.Reset()
		h.Unlock()
		h.release()
		err = h.conn.Close()
	})
	return err
}

func (h *httpTransportSocket) LocalAddr() net.Addr {
	return h.conn.LocalAddr()
}

func (h *httpTransportSocket) RemoteAddr() net.Addr {
	return h.conn.RemoteAddr()
}

//////////////////////////////////////////////
// http transport listener
//////////////////////////////////////////////

type httpTransportListener struct {
	ht       *httpTransport
	listener net.Listener
	sem      chan struct{}
}

func initHttpTransportListener(ht *httpTransport, listener net.Listener) *httpTransportListener {
	return &httpTransportListener{
		ht:       ht,
		listener: listener,
		// 此处sizeof(struct{}{})为0，所以尽管DefaultMAXConnNum数字很大，但是不占用空间
		sem: make(chan struct{}, DefaultMAXConnNum),
	}
}

func (h *httpTransportListener) acquire() { h.sem <- struct{}{} }
func (h *httpTransportListener) release() { <-h.sem }

func (h *httpTransportListener) Addr() string {
	return h.listener.Addr().String()
}

func (h *httpTransportListener) Close() error {
	return h.listener.Close()
}

func (h *httpTransportListener) Accept(fn func(Socket)) error {
	var (
		err      error
		c        net.Conn
		ok       bool
		ne       net.Error
		tmpDelay time.Duration
	)

	for {
		h.acquire() // 若connect chan已满,则会阻塞在此处
		c, err = h.listener.Accept()
		if err != nil {
			h.release()
			if ne, ok = err.(net.Error); ok && ne.Temporary() {
				if tmpDelay == 0 {
					tmpDelay = 5 * time.Millisecond
				} else {
					tmpDelay *= 2
				}
				// if max := 1 * time.Second; tempDelay > max {
				if tmpDelay > DefaultMaxSleepTime {
					tmpDelay = DefaultMaxSleepTime
				}
				log.Info("http: Accept error: %v; retrying in %v\n", err, tmpDelay)
				time.Sleep(tmpDelay)
				continue
			}
			return err
		}
		// if tcpConn, ok = c.(net.TCPConn); ok {
		// 	// tcpConn.SetReadBuffer(DefaultTcpReadBufferSize) // use the os default value
		// 	// tcpConn.SetWriteBuffer(DefaultTcpReadBufferSize) // use the os default value
		// }

		sock := initHttpTransportSocket(h.ht, c, h.release)

		// 逻辑执行再单独启动一个goroutine
		go func() {
			defer func() {
				if r := recover(); r != nil {
					const size = 64 << 10
					buf := make([]byte, size)
					buf = buf[:runtime.Stack(buf, false)]
					log.Error("http: panic serving %v: %v\n%s", c.RemoteAddr(), r, buf)
					sock.Close() // 遇到错误退出的时候保证socket fd的回收
				}
			}()

			fn(sock) // rpcServer:handlePkg 函数里面有一个defer语句段，保证了正常退出的情况下sock.Close()
		}()
	}
}

//////////////////////////////////////////////
// http transport
//////////////////////////////////////////////

type httpTransport struct {
	opts Options
}

func (h *httpTransport) Dial(addr string, opts ...DialOption) (Client, error) {
	dopts := DialOptions{
		Timeout: DefaultDialTimeout,
	}

	for _, opt := range opts {
		opt(&dopts)
	}

	var (
		conn net.Conn
		err  error
	)
	conn, err = net.DialTimeout("tcp", addr, dopts.Timeout)
	if err != nil {
		return nil, err
	}

	return initHttpTransportClient(h, addr, conn, dopts), nil
}

func listen(addr string, fn func(string) (net.Listener, error)) (net.Listener, error) {
	var (
		err      error
		ln       net.Listener
		min, max int
	)

	// host:port || host:min-max
	parts := strings.Split(addr, ":")

	//
	if len(parts) < 2 {
		return fn(addr)
	}

	// try to extract port range
	ports := strings.Split(parts[len(parts)-1], "-")

	// single port
	// 单个port
	if len(ports) < 2 {
		return fn(addr)
	}

	// we have a port range

	// extract min port
	min, err = strconv.Atoi(ports[0])
	if err != nil {
		return nil, errors.New("unable to extract port range")
	}

	// extract max port
	max, err = strconv.Atoi(ports[1])
	if err != nil {
		return nil, errors.New("unable to extract port range")
	}

	// set host
	host := parts[:len(parts)-1]

	// range the ports
	// 遍历一个port range，以找到一个可以bind的port
	for port := min; port <= max; port++ {
		// try bind to host:port
		ln, err = fn(fmt.Sprintf("%s:%d", host, port))
		if err == nil {
			return ln, nil // 找到之后就退出
		}

		// hit max port
		if port == max {
			return nil, err
		}
	}

	// why are we here?
	// 仅仅是为了满足编译器检查错误需求(所有分支都有返回)
	return nil, fmt.Errorf("unable to bind to %s", addr)
}

func (h *httpTransport) Listen(addr string, opts ...ListenOption) (Listener, error) {
	var options ListenOptions
	for _, o := range opts {
		o(&options)
	}

	var (
		l   net.Listener
		err error
	)

	fn := func(addr string) (net.Listener, error) {
		return net.Listen("tcp", addr)
	}

	l, err = listen(addr, fn)

	if err != nil {
		return nil, err
	}

	return initHttpTransportListener(h, l), nil
}

func (h *httpTransport) String() string {
	return "http"
}

func newHTTPTransport(opts ...Option) *httpTransport {
	var options Options
	for _, o := range opts {
		o(&options)
	}
	return &httpTransport{opts: options}
}

func NewHTTPTransport(opts ...Option) Transport {
	return newHTTPTransport(opts...)
}
