/******************************************************
# DESC    : application configure
# AUTHOR  : Alex Stocks
# VERSION : 1.0
# LICENCE : Apache Licence 2.0
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2016-06-08 18:00
# FILE    : applicaton_config.go
******************************************************/

package common

import (
	"fmt"
)

type ApplicationConfig struct {
	// 组织名(BU或部门)
	Organization string
	// 应用名称
	Name string
	// 模块名称
	Module string
	// 模块版本
	Version string
	// 应用负责人
	Owner string
}

func (this *ApplicationConfig) ToString() string {
	return fmt.Sprintf("ApplicationConfig is {name:%s, version:%s, owner:%s, module:%s, organization:%s}",
		this.Name, this.Version, this.Owner, this.Module, this.Organization)
}
