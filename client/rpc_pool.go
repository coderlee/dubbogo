/******************************************************
# DESC    :
# AUTHOR  : Alex Stocks
# VERSION : 1.0
# LICENCE : Apache Licence 2.0
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2016-06-30 10:45
# FILE    : rpc_pool.go
******************************************************/

package client

import (
	"fmt"
	"sync"
	"time"
)

import (
	"github.com/AlexStocks/dubbogo/transport"
)

type pool struct {
	size int   // 从line 92可见，size是[]*poolConn数组的size
	ttl  int64 // 从line 61 可见，ttl是每个poolConn的有效期时间. pool对象会在getConn时执行ttkl检查

	sync.Mutex
	conns map[string][]*poolConn // 从[]*poolConn 可见key是连接地址，而value是对应这个地址的连接数组
}

type poolConn struct {
	once *sync.Once
	transport.Client
	created int64 // 为0，则说明没有被创建或者被销毁了
}

func newPool(size int, ttl time.Duration) *pool {
	return &pool{
		size:  size,
		ttl:   int64(ttl.Seconds()),
		conns: make(map[string][]*poolConn),
	}
}

func (p *poolConn) Close() error {
	// log.Debug("close poolConn{%#v}", p)
	var err error = fmt.Errorf("close poolConn{%#v} again", p)
	p.once.Do(func() {
		p.Client.Close()
		p.created = 0
		err = nil
	})
	return err
}

func (p *pool) getConn(addr string, tr transport.Transport, opts ...transport.DialOption) (*poolConn, error) {
	p.Lock()
	conns := p.conns[addr]
	now := time.Now().Unix()

	// while we have conns check age and then return one
	// otherwise we'll create a new conn
	for len(conns) > 0 {
		conn := conns[len(conns)-1]
		conns = conns[:len(conns)-1] // 非常好的删除最后一个element的技巧
		p.conns[addr] = conns

		// if conn is old kill it and move on
		if d := now - conn.created; d > p.ttl {
			conn.Client.Close()
			continue
		}

		// we got a good conn, lets unlock and return it
		p.Unlock()

		return conn, nil
	}

	p.Unlock()

	// create new conn
	// Dial函数是DefaultTransport.Dial，而DefaultTransport = newHTTPTransport()，
	// 所以返回的client实际是httpTransportClient
	c, err := tr.Dial(addr, opts...)
	if err != nil {
		return nil, err
	}
	return &poolConn{&sync.Once{}, c, time.Now().Unix()}, nil
}

func (p *pool) release(addr string, conn *poolConn, err error) {
	if conn == nil || conn.created == 0 {
		return
	}
	// don't store the conn if it has errored
	if err != nil {
		conn.Close() // 须经过(poolConn)Close，以防止多次close transport client
		return
	}

	// otherwise put it back for reuse
	p.Lock()
	conns := p.conns[addr]
	if len(conns) >= p.size {
		p.Unlock()
		conn.Client.Close()
		return
	}
	p.conns[addr] = append(conns, conn)
	p.Unlock()
}
