/******************************************************
# DESC    : service url unit test
# AUTHOR  : Alex Stocks
# VERSION : 1.0
# LICENCE : Apache Licence 2.0
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2016-06-14 16:58
# FILE    : service_url_test.go
******************************************************/

// package registry_test // for go test -v service_url_test.go
package registry // for go test -v

import (
	"fmt"
	"testing"
)

func TestServiceURL_NewServiceURL(t *testing.T) {
	var (
		err error
		url = "dubbo://116.211.15.190:20880/im.youni.weboa.common.service.IRegisterService?anyhost=true&application=weboa&dubbo=2.5.3&interface=im.youni.weboa.common.service.IRegisterService&methods=registerUser,exists&pid=13772&revision=1.2.2&side=provider&timestamp=1464255871323"
		// serviceURL *common.ServiceURL
		serviceURL *ServiceURL
	)

	// serviceURL, err = common.NewServiceURL(url)
	serviceURL, err = NewServiceURL(url)
	if err != nil {
		t.Errorf("ServiceUrl.Init(url{%s}) = %v", url, err)
	}

	/*
		serviceUrl{&common.ServiceURL{Protocol:"dubbo", Location:"116.211.15.190:20880", Path:"/im.youni.weboa.common.service.IRegisterService", Ip:"116.211.15.190", Port:"20880", Version:"", Group:"", Query:url.Values{"anyhost":[]string{"true"}, "application":[]string{"weboa"}, "interface":[]string{"im.youni.weboa.common.service.IRegisterService"}, "methods":[]string{"registerUser,exists"}, "pid":[]string{"13772"}, "revision":[]string{"1.2.2"}, "dubbo":[]string{"2.5.3"}, "side":[]string{"provider"}, "timestamp":[]string{"1464255871323"}}, :""}}
		serviceURL.protocol: dubbo
		serviceURL.location: 116.211.15.190:20880
		serviceURL.path: /im.youni.weboa.common.service.IRegisterService
		serviceURL.ip: 116.211.15.190
		serviceURL.port: 20880
		serviceURL.version:
		serviceURL.group:
		serviceURL.query: map[anyhost:[true] application:[weboa] interface:[im.youni.weboa.common.service.IRegisterService] methods:[registerUser,exists] pid:[13772] revision:[1.2.2] dubbo:[2.5.3] side:[provider] timestamp:[1464255871323]]
		serviceURL.query.interface: im.youni.weboa.common.service.IRegisterService
	*/
	fmt.Printf("serviceUrl{%#v}\n", serviceURL)
	fmt.Println("serviceURL.protocol:", serviceURL.Protocol)
	fmt.Println("serviceURL.location:", serviceURL.Location)
	fmt.Println("serviceURL.path:", serviceURL.Path)
	fmt.Println("serviceURL.ip:", serviceURL.Ip)
	fmt.Println("serviceURL.port:", serviceURL.Port)
	fmt.Println("serviceURL.version:", serviceURL.Version)
	fmt.Println("serviceURL.group:", serviceURL.Group)
	fmt.Println("serviceURL.query:", serviceURL.Query)
	fmt.Println("serviceURL.query.interface:", serviceURL.Query.Get("interface"))
}
