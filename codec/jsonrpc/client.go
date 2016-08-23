package jsonrpc

import (
	"encoding/json"
	"io"
	"reflect"
	"sync"
)

import (
	log "github.com/AlexStocks/log4go"
	// "github.com/gorilla/rpc/v2/json2"
)

import (
	"fmt"
	"github.com/AlexStocks/dubbogo/codec"
)

const (
	MAX_JSONRPC_ID = 0x7FFFFFFF
)

type clientCodec struct {
	dec *json.Decoder // for reading JSON values
	enc *json.Encoder // for writing JSON values
	c   io.Closer

	// temporary work space
	req  clientRequest
	resp clientResponse

	sync.Mutex
	pending map[uint64]string
}

// type clientRequest struct {
// 	Method string         `json:"method"`
// 	Params [1]interface{} `json:"params"`
// 	ID     uint64         `json:"id"`
// }

type clientRequest struct {
	Version string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      uint64      `json:"id"`
}

// type clientResponse struct {
// 	ID     uint64           `json:"id"`
// 	Result *json.RawMessage `json:"result"`
// 	Error  interface{}      `json:"error"`
// }

type clientResponse struct {
	Version string           `json:"jsonrpc"`
	ID      uint64           `json:"id"`
	Result  *json.RawMessage `json:"result,omitempty"`
	// Error   *json.RawMessage `json:"error"`
	Error *Error `json:"error,omitempty"`
}

func (r *clientResponse) reset() {
	r.Version = ""
	r.ID = 0
	r.Result = nil
	r.Error = nil
}

func newClientCodec(conn io.ReadWriteCloser) *clientCodec {
	return &clientCodec{
		dec:     json.NewDecoder(conn),
		enc:     json.NewEncoder(conn),
		c:       conn,
		pending: make(map[uint64]string),
	}
}

func (c *clientCodec) Write(m *codec.Message, param interface{}) error {
	// If return error: it will be returned as is for this call.
	// Allow param to be only Array, Slice, Map or Struct.
	// When param is nil or uninitialized Map or Slice - omit "params".
	if param != nil {
		switch k := reflect.TypeOf(param).Kind(); k {
		case reflect.Map:
			if reflect.TypeOf(param).Key().Kind() == reflect.String {
				if reflect.ValueOf(param).IsNil() {
					param = nil
				}
			}
		case reflect.Slice:
			if reflect.ValueOf(param).IsNil() {
				param = nil
			}
		case reflect.Array, reflect.Struct:
		case reflect.Ptr:
			switch k := reflect.TypeOf(param).Elem().Kind(); k {
			case reflect.Map:
				if reflect.TypeOf(param).Elem().Key().Kind() == reflect.String {
					if reflect.ValueOf(param).Elem().IsNil() {
						param = nil
					}
				}
			case reflect.Slice:
				if reflect.ValueOf(param).Elem().IsNil() {
					param = nil
				}
			case reflect.Array, reflect.Struct:
			default:
				return NewError(errInternal.Code, "unsupported param type: Ptr to "+k.String())
			}
		default:
			return NewError(errInternal.Code, "unsupported param type: "+k.String())
		}
	}

	c.req.Version = "2.0"
	c.req.Method = m.Method
	// c.req.Params = b
	c.req.Params = param
	c.req.ID = m.Id & MAX_JSONRPC_ID
	c.Lock()
	// c.pending[m.Id] = m.Method // 此处如果用m.Id会导致error: can not find method of response id 280698512
	c.pending[c.req.ID] = m.Method
	c.Unlock()

	return c.enc.Encode(&c.req)
}

// func (r *clientResponse) reset() {
// 	r.ID = 0
// 	r.Result = nil
// 	r.Error = nil
// }

func (c *clientCodec) ReadHeader(m *codec.Message) error {
	c.resp.reset()
	if err := c.dec.Decode(&c.resp); err != nil {
		if err == io.EOF {
			log.Debug("c.dec.Decode(c.resp{%v}) = err{%T-%v}, err == io.EOF", c.resp, err, err)
			return err
		}
		log.Debug("c.dec.Decode(c.resp{%v}) = err{%T-%v}, err != io.EOF", c.resp, err, err)
		return NewError(errInternal.Code, err.Error())
	}
	// if c.resp.ID == nil {
	// 	return c.resp.Error
	// }

	var ok bool
	c.Lock()
	m.Method, ok = c.pending[c.resp.ID]
	if !ok {
		c.Unlock()
		err := fmt.Errorf("can not find method of response id %v, response error:%v", c.resp.ID, c.resp.Error)
		log.Debug("clientCodec.ReadHeader(@m{%v}) = error{%v}", m, err)
		return err
	}
	delete(c.pending, c.resp.ID)
	c.Unlock()

	m.Error = ""
	m.Id = c.resp.ID
	if c.resp.Error != nil {
		// x, ok := c.resp.Error.(string)
		// if !ok {
		// 	return fmt.Errorf("invalid error %v", c.resp.Error)
		// }
		// if x == "" {
		// 	x = "unspecified error"
		// }
		// m.Error = x
		m.Error = c.resp.Error.Error()
	}

	return nil
}

func (c *clientCodec) ReadBody(x interface{}) error {
	if x == nil || c.resp.Result == nil {
		return nil
	}
	return json.Unmarshal(*c.resp.Result, x)
	// if err := json.Unmarshal(*c.resp.Result, x); err != nil {
	// 	e := NewError(errInternal.Code, err.Error())
	// 	e.Data = NewError(errInternal.Code, "some other Call failed to unmarshal Reply")
	// 	return e
	// }
	// return nil
}

func (c *clientCodec) Close() error {
	return c.c.Close()
}
