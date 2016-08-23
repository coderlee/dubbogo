/******************************************************
# DESC    : service & service url
# AUTHOR  : Alex Stocks
# VERSION : 1.0
# LICENCE : Apache Licence 2.0
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2016-06-08 16:40
# FILE    : service_url.go
******************************************************/

package registry

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

//////////////////////////////////////////
// service url event type
//////////////////////////////////////////

type ServiceURLEventType int

const (
	ServiceURLAdd = iota
	ServiceURLDel
	ServiceURLUpdate
)

var serviceURLEventTypeStrings = [...]string{
	"add service url",
	"delete service url",
	"updaet service url",
}

func (t ServiceURLEventType) String() string {
	return serviceURLEventTypeStrings[t]
}

//////////////////////////////////////////
// service url event
//////////////////////////////////////////

type ServiceURLEvent struct {
	Action  ServiceURLEventType
	Service *ServiceURL
}

func (e ServiceURLEvent) String() string {
	return fmt.Sprintf("ServiceURLEvent{Action{%s}, Service{%#v}}", e.Action.String(), e.Service)
}

//////////////////////////////////////////
// service url
//////////////////////////////////////////

type ServiceURL struct {
	Protocol     string
	Location     string // ip+port
	Path         string // like  /com.qianmi.dubbo.UserProvider
	Ip           string
	Port         string
	Version      string
	Group        string
	Query        url.Values
	Weight       int32
	PrimitiveURL string
}

func NewServiceURL(urlString string) (*ServiceURL, error) {
	var (
		err          error
		rawUrlString string
		serviceUrl   *url.URL
		this         = &ServiceURL{}
	)

	rawUrlString, err = url.QueryUnescape(urlString)
	if err != nil {
		return nil, fmt.Errorf("url.QueryUnescape(%s),  error{%v}", urlString, err)
	}

	serviceUrl, err = url.Parse(rawUrlString)
	if err != nil {
		return nil, fmt.Errorf("url.Parse(url string{%s}),  error{%v}", rawUrlString, err)
	}

	this.Query, err = url.ParseQuery(serviceUrl.RawQuery)
	if err != nil {
		return nil, fmt.Errorf("url.ParseQuery(raw url string{%s}),  error{%v}", serviceUrl.RawQuery, err)
	}

	this.PrimitiveURL = urlString
	this.Protocol = serviceUrl.Scheme
	this.Location = serviceUrl.Host
	this.Path = serviceUrl.Path
	if strings.Contains(this.Location, ":") {
		this.Ip, this.Port, err = net.SplitHostPort(this.Location)
		if err != nil {
			return nil, fmt.Errorf("net.SplitHostPort(Url.Host{%s}), error{%v}", this.Location, err)
		}
	}
	this.Group = this.Query.Get("group")
	this.Version = this.Query.Get("version")

	return this, nil
}

func (this *ServiceConfig) ServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		Protocol: this.Protocol,
		Service:  this.Service,
		Group:    this.Group,
		Version:  this.Version,
	}
}

func (this *ServiceURL) CheckMethod(method string) bool {
	var (
		methodArray []string
	)

	methodArray = strings.Split(this.Query.Get("methods"), ",")
	for _, m := range methodArray {
		if m == method {
			return true
		}
	}

	return false
}
