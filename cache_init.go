package cache

import (
	"github.com/injoyai/base/maps"
	"github.com/injoyai/conv"
)

var DefaultDir = "./data/cache/"

// NewFile 新建文件缓存,默认名称"_default"
// 万次读写速度4.18秒
// 万次协程读写速度2.21秒
func NewFile(name string, tags ...string) *File {
	tag := conv.GetDefaultString("var", tags...)
	return newFile(name, tag)
}

// NewCycle 新建内存缓存,存储记录,循环使用
// 百万次读写速度4.31秒
// 百万次携程读写速度3.76秒
func NewCycle(num int) *Cycle {
	return newCycle(num)
}

// NewMap 新建内存缓存
// 百万次读写速度1.2秒
// 百万次携程读写速度1.25秒
func NewMap() *maps.Safe {
	return maps.NewSafe()
}
