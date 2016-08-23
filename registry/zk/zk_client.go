/******************************************************
# DESC    : zookeeper client
# AUTHOR  : Alex Stocks
# VERSION : 1.0
# LICENCE : Apache Licence 2.0
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2016-07-06 10:23
# FILE    : zk_client.go
******************************************************/

package zookeeper

import (
	"errors"
	"fmt"
	"path"
	"strings"
	"sync"
)

import (
	log "github.com/AlexStocks/log4go"
	"github.com/samuel/go-zookeeper/zk"
)

import (
	"github.com/AlexStocks/dubbogo/common"
)

var (
	ZK_CLIENT_CONN_NIL_ERR = errors.New("zookeeperclient{conn} is nil")
)

type zookeeperClient struct {
	name          string
	zkAddrs       []string
	sync.Mutex             // for conn
	conn          *zk.Conn // 这个conn不能被close两次，否则会收到 “panic: close of closed channel”
	timeout       int
	exit          chan struct{}
	wait          sync.WaitGroup
	eventRegistry map[string][]*chan struct{}
}

func stateToString(state zk.State) string {
	switch state {
	case zk.StateDisconnected:
		return "zookeeper disconnected"
	case zk.StateConnecting:
		return "zookeeper connecting"
	case zk.StateAuthFailed:
		return "zookeeper auth failed"
	case zk.StateConnectedReadOnly:
		return "zookeeper connect readonly"
	case zk.StateSaslAuthenticated:
		return "zookeeper sasl authenticaed"
	case zk.StateExpired:
		return "zookeeper connection expired"
	case zk.StateConnected:
		return "zookeeper conneced"
	case zk.StateHasSession:
		return "zookeeper has session"
	case zk.StateUnknown:
		return "zookeeper unknown state"
	case zk.State(zk.EventNodeDeleted):
		return "zookeeper node deleted"
	case zk.State(zk.EventNodeDataChanged):
		return "zookeeper node data changed"
	default:
		return state.String()
	}

	return "zookeeper unknown state"
}

func newZookeeperClient(name string, zkAddrs []string, timeout int) (*zookeeperClient, error) {
	var (
		err   error
		event <-chan zk.Event
		this  *zookeeperClient
	)

	this = &zookeeperClient{
		name:          name,
		zkAddrs:       zkAddrs,
		timeout:       timeout,
		exit:          make(chan struct{}),
		eventRegistry: make(map[string][]*chan struct{}),
	}
	// connect to zookeeper
	this.conn, event, err = zk.Connect(zkAddrs, common.TimeSecondDuration(timeout))
	if err != nil {
		return nil, err
	}

	this.wait.Add(1)
	go this.handleZkEvent(event)

	return this, nil
}

func (this *zookeeperClient) handleZkEvent(session <-chan zk.Event) {
	var (
		state int
		event zk.Event
	)

	defer func() {
		this.wait.Done()
		log.Info("zk{path:%v, name:%s} connection goroutine game over.", this.zkAddrs, this.name)
	}()

LOOP:
	for {
		select {
		case <-this.exit:
			break LOOP
		case event = <-session:
			log.Warn("client{%s} get a zookeeper event{type:%s, server:%s, path:%s, state:%d-%s, err:%s}",
				this.name, event.Type.String(), event.Server, event.Path, event.State, stateToString(event.State), event.Err)
			switch (int)(event.State) {
			case (int)(zk.StateDisconnected):
				log.Warn("zk{addr:%s} state is StateDisconnected, so close the zk client{name:%s}.", this.zkAddrs, this.name)
				this.stop()
				this.Lock()
				if this.conn != nil {
					this.conn.Close()
					this.conn = nil
				}
				this.Unlock()
				break LOOP
			case (int)(zk.EventNodeDataChanged), (int)(zk.EventNodeChildrenChanged):
				log.Info("zkClient{%s} get zk node changed event{path:%s}", this.name, event.Path)
				this.Lock()
				for p, a := range this.eventRegistry {
					if strings.HasPrefix(p, event.Path) {
						log.Info("send event{state:zk.EventNodeDataChange, Path:%s} notify event to path{%s} related watcher", event.Path, p)
						for _, e := range a {
							*e <- struct{}{}
						}
					}
				}
				this.Unlock()
			case (int)(zk.StateConnecting), (int)(zk.StateConnected), (int)(zk.StateHasSession):
				if state != (int)(zk.StateConnecting) || state != (int)(zk.StateDisconnected) {
					continue
				}
				if a, ok := this.eventRegistry[event.Path]; ok && 0 < len(a) {
					for _, e := range a {
						*e <- struct{}{}
					}
				}
			}
			state = (int)(event.State)
		}
	}
}

func (this *zookeeperClient) registerEvent(zkPath string, event *chan struct{}) {
	if zkPath == "" || event == nil {
		return
	}

	this.Lock()
	a := this.eventRegistry[zkPath]
	a = append(a, event)
	this.eventRegistry[zkPath] = a
	log.Debug("zkClient{%s} register event{path:%s, ptr:%p}", this.name, zkPath, event)
	this.Unlock()
}

func (this *zookeeperClient) unregisterEvent(zkPath string, event *chan struct{}) {
	if zkPath == "" {
		return
	}

	this.Lock()
	for {
		a, ok := this.eventRegistry[zkPath]
		if !ok {
			break
		}
		for i, e := range a {
			if e == event {
				arr := a
				a = append(arr[:i], arr[i+1:]...)
				log.Debug("zkClient{%s} unregister event{path:%s, event:%p}", this.name, zkPath, event)
			}
		}
		log.Debug("after zkClient{%s} unregister event{path:%s, event:%p}, array length %d", this.name, zkPath, event, len(a))
		if len(a) == 0 {
			delete(this.eventRegistry, zkPath)
		} else {
			this.eventRegistry[zkPath] = a
		}
		break
	}
	this.Unlock()
}

func (this *zookeeperClient) done() <-chan struct{} {
	return this.exit
}

func (this *zookeeperClient) stop() bool {
	select {
	case <-this.exit:
		return true
	default:
		close(this.exit)
	}

	return false
}

func (this *zookeeperClient) zkConnValid() bool {
	select {
	case <-this.exit:
		return false
	default:
	}

	var valid bool = true
	this.Lock()
	if this.conn == nil {
		valid = false
	}
	this.Unlock()

	return valid
}

func (this *zookeeperClient) Close() {
	this.stop()
	this.wait.Wait()
	this.Lock()
	if this.conn != nil {
		this.conn.Close() // 等着所有的goroutine退出后，再关闭连接
		this.conn = nil
	}
	this.Unlock()
	log.Warn("zkClient{name:%s, zk addr:%s} exit now.", this.name, this.zkAddrs)
}

// 节点须逐级创建
func (this *zookeeperClient) Create(basePath string) error {
	var (
		err     error
		tmpPath string
	)

	log.Debug("zookeeperClient.Create(basePath{%s})", basePath)
	for _, str := range strings.Split(basePath, "/")[1:] {
		tmpPath = path.Join(tmpPath, "/", str)
		// log.Debug("create zookeeper path: \"%s\"\n", tmpPath)
		err = ZK_CLIENT_CONN_NIL_ERR
		this.Lock()
		if this.conn != nil {
			_, err = this.conn.Create(tmpPath, []byte(""), 0, zk.WorldACL(zk.PermAll))
		}
		this.Unlock()
		if err != nil {
			if err == zk.ErrNodeExists {
				log.Error("zk.create(\"%s\") exists\n", tmpPath)
			} else {
				log.Error("zk.create(\"%s\") error(%v)\n", tmpPath, err)
				return err
			}
		}
	}

	return nil
}

// 像创建一样，删除节点的时候也只能从叶子节点逐级回退删除
// 当节点还有子节点的时候，删除是不会成功的
func (this *zookeeperClient) Delete(basePath string) error {
	var (
		err error
	)

	err = ZK_CLIENT_CONN_NIL_ERR
	this.Lock()
	if this.conn != nil {
		err = this.conn.Delete(basePath, -1)
	}
	this.Unlock()

	return err
}

func (this *zookeeperClient) RegisterTemp(basePath string, node string) (string, error) {
	var (
		err     error
		data    []byte
		zkPath  string
		tmpPath string
	)

	err = ZK_CLIENT_CONN_NIL_ERR
	data = []byte("")
	zkPath = path.Join(basePath) + "/" + node
	this.Lock()
	if this.conn != nil {
		tmpPath, err = this.conn.Create(zkPath, data, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
	}
	this.Unlock()
	// log.Debug("zookeeperClient.RegisterTemp(basePath{%s}) = tempPath{%s}", zkPath, tmpPath)
	if err != nil {
		log.Error("conn.Create(\"%s\", zk.FlagEphemeral) = error(%v)\n", zkPath, err)
		// if err != zk.ErrNodeExists {
		return "", err
		// }
	}
	log.Debug("zkClient{%s} create a temp zookeeper node:%s\n", this.name, tmpPath)
	return tmpPath, nil
}

func (this *zookeeperClient) RegisterTempSeq(basePath string, data []byte) (string, error) {
	var (
		err     error
		tmpPath string
	)

	err = ZK_CLIENT_CONN_NIL_ERR
	this.Lock()
	if this.conn != nil {
		tmpPath, err = this.conn.Create(path.Join(basePath)+"/", data, zk.FlagEphemeral|zk.FlagSequence, zk.WorldACL(zk.PermAll))
	}
	this.Unlock()
	log.Debug("zookeeperClient.RegisterTempSeq(basePath{%s}) = tempPath{%s}", basePath, tmpPath)
	if err != nil {
		log.Error("zkClient{%s} conn.Create(\"%s\", \"%s\", zk.FlagEphemeral|zk.FlagSequence) error(%v)\n",
			this.name, basePath, string(data), err)
		// if err != zk.ErrNodeExists {
		return "", err
		// }
	}
	log.Debug("zkClient{%s} create a temp zookeeper node:%s\n", this.name, tmpPath)
	return tmpPath, nil
}

func (this *zookeeperClient) getChildrenW(path string) ([]string, <-chan zk.Event, error) {
	var (
		err      error
		children []string
		stat     *zk.Stat
		watch    <-chan zk.Event
	)

	err = ZK_CLIENT_CONN_NIL_ERR
	this.Lock()
	if this.conn != nil {
		children, stat, watch, err = this.conn.ChildrenW(path)
	}
	this.Unlock()
	if err != nil {
		if err == zk.ErrNoNode {
			return nil, nil, fmt.Errorf("path{%s} has none children", path)
		}
		log.Error("zk.ChildrenW(path{%s}) = error(%v)", path, err)
		return nil, nil, err
	}
	if stat == nil {
		return nil, nil, fmt.Errorf("path{%s} has none children", path)
	}
	if len(children) == 0 {
		return nil, nil, fmt.Errorf("path{%s} has none children", path)
	}

	return children, watch, nil
}

func (this *zookeeperClient) getChildren(path string) ([]string, error) {
	var (
		err      error
		children []string
		stat     *zk.Stat
	)

	err = ZK_CLIENT_CONN_NIL_ERR
	this.Lock()
	if this.conn != nil {
		children, stat, err = this.conn.Children(path)
	}
	this.Unlock()
	if err != nil {
		if err == zk.ErrNoNode {
			return nil, fmt.Errorf("path{%s} has none children", path)
		}
		log.Error("zk.Children(path{%s}) = error(%v)", path, err)
		return nil, err
	}
	if stat == nil {
		return nil, fmt.Errorf("path{%s} has none children", path)
	}
	if len(children) == 0 {
		return nil, fmt.Errorf("path{%s} has none children", path)
	}

	return children, nil
}

func (this *zookeeperClient) existW(zkPath string) (<-chan zk.Event, error) {
	var (
		exist bool
		err   error
		watch <-chan zk.Event
	)

	err = ZK_CLIENT_CONN_NIL_ERR
	this.Lock()
	if this.conn != nil {
		exist, _, watch, err = this.conn.ExistsW(zkPath)
	}
	this.Unlock()
	if err != nil {
		log.Error("zkClient{%s}.ExistsW(path{%s}) = error{%v}.", this.name, zkPath, err)
		return nil, err
	}
	if !exist {
		log.Warn("zkClient{%s}'s App zk path{%s} does not exist.", this.name, zkPath)
		return nil, fmt.Errorf("zkClient{%s} App zk path{%s} does not exist.", this.name, zkPath)
	}

	return watch, nil
}
