/******************************************************
# DESC    : network type & codec type
# AUTHOR  : Alex Stocks
# VERSION : 1.0
# LICENCE : Apache Licence 2.0
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2016-06-22 13:20
# FILE    : const.go
******************************************************/

package codec

//////////////////////////////////////////
// transport network
//////////////////////////////////////////

type NetworkType int

const (
	NETWORK_TYPE_BEGIN NetworkType = iota
	TCP
	HTTP
	NETWORK_TYPE_END
)

var networkTypeStrings = [...]string{
	"NETWORK_TYPE_BEGIN",
	"TCP",
	"HTTP",
	"NETWORK_TYPE_END",
}

func (this NetworkType) String() string {
	if NETWORK_TYPE_BEGIN < this && this < NETWORK_TYPE_END {
		return networkTypeStrings[this]
	}

	return ""
}

//////////////////////////////////////////
// codec type
//////////////////////////////////////////

type CodecType int

const (
	JSONRPC CodecType = iota
)

var codecTypeStrings = [...]string{
	"jsonrpc",
}

func (c CodecType) String() string {
	return codecTypeStrings[c]
}
