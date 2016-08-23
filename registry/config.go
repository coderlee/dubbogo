/******************************************************
# DESC    : registry config
# AUTHOR  : Alex Stocks
# VERSION : 1.0
# LICENCE : Apache Licence 2.0
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2016-06-19 11:01
# FILE    : config.go
******************************************************/

package registry

import (
	"fmt"
	"strconv"
)

type RegistryConfig struct {
	Address  []string `required:"true"`
	UserName string
	Password string
	Timeout  int `default:"5"` // unit: second
}

type ServiceConfigIf interface {
	String() string
	ServiceEqual(url *ServiceURL) bool
}

/*
 * 这个struct不可包含RegistryConfig
 *
 * zookeeperRegistry已经包含了RegistryConfig,而ConsumerzookeeperRegistry定义如下
 * type consumerZookeeperRegistry struct {
 *		*zookeeperRegistry
 *  	services []ServiceConf
 * }
 * 如果ConsumerRegistryConfig包含RegistryConfig,则有如下重复定义:
 * consumerZookeeperRegistry.zookeeperRegistry.RegistryConfig
 * consumerZookeeperRegistry.services[0].RegistryConfig
 */

type ServiceConfig struct {
	Protocol string `required:"true",default:"dubbo"` // codec string, jsonrpc etc
	// func (this *consumerZookeeperRegistry) Register(conf ServiceConfig) 函数用到了Service
	// "/dubbo/com.ofpay.demo.api.UserProvider/providers" Service指代中间这一部分，目前假设它与interface相同
	Service string `required:"true"`
	Group   string
	Version string
}

func (this ServiceConfig) String() string {
	return fmt.Sprintf("%s@%s-%s-%s", this.Service, this.Protocol, this.Group, this.Version)
}

// 目前不支持一个service的多个协议的使用，将来如果要支持，关键点就是修改这个函数
func (this ServiceConfig) ServiceEqual(url *ServiceURL) bool {
	if this.Protocol != url.Protocol {
		return false
	}

	if this.Service != url.Query.Get("interface") {
		return false
	}

	if this.Group != url.Group {
		return false
	}

	if this.Version != url.Version {
		return false
	}

	return true
}

type ProviderServiceConfig struct {
	ServiceConfig
	Path    string
	Methods string
}

func (this ProviderServiceConfig) String() string {
	return fmt.Sprintf("%s@%s-%s-%s-%s/%s", this.Service, this.Protocol, this.Group, this.Version, this.Path, this.Methods)
}

func (this ProviderServiceConfig) ServiceEqual(url *ServiceURL) bool {
	if this.Protocol != url.Protocol {
		return false
	}

	if this.Service != url.Query.Get("interface") {
		return false
	}

	if this.Group != url.Group {
		return false
	}

	if this.Version != url.Version {
		return false
	}

	if this.Path != url.Path {
		return false
	}

	if this.Methods != url.Query.Get("methods") {
		return false
	}

	return true
}

type ServerConfig struct {
	Protocol string `required:"true",default:"dubbo"` // codec string, jsonrpc etc
	IP       string
	Port     int `required:"true"`
}

func (this *ServerConfig) Address() string {
	return this.IP + ":" + strconv.Itoa(this.Port)
}
