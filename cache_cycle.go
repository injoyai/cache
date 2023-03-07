package cache

import (
	json "github.com/json-iterator/go"
	"strconv"
	"sync"
	"time"
)

type CycleChan struct {
	key   string           //标识,自动生成
	C     chan interface{} //数据通道
	close func()           //关闭函数
}

func (this *CycleChan) Close() error {
	if this.close != nil {
		this.close()
		this.close = nil
	}
	return nil
}

// Cycle 固定列表长度,循环使用
type Cycle struct {
	key    string                //标识,手动设置
	list   []interface{}         //列表数据
	offset int                   //当前数据位置下标
	length int                   //列表长度
	cycle  bool                  //循环使用,数据量已经超过列表的长度,覆盖了老数据
	mChan  map[string]*CycleChan //监听数据 todo 优化成列表
	mu     sync.RWMutex          //锁
}

// SetKey 设置标识
func (this *Cycle) SetKey(key string) *Cycle {
	this.key = key
	return this
}

// GetKey 获取标识
func (this *Cycle) GetKey() string {
	return this.key
}

func (this *Cycle) newChan() *CycleChan {
	c := &CycleChan{
		key: strconv.Itoa(int(time.Now().UnixNano())),
		C:   make(chan interface{}),
	}
	c.close = func() {
		this.mu.Lock()
		delete(this.mChan, c.key)
		this.mu.Unlock()
		close(c.C)
	}
	return c
}

// Chan 开启一个监听数据的通道
func (this *Cycle) Chan() *CycleChan {
	c := this.newChan()
	this.mu.RLock()
	_, ok := this.mChan[c.key]
	this.mu.RUnlock()
	if ok {
		return this.Chan()
	}
	this.mu.Lock()
	this.mChan[c.key] = c
	this.mu.Unlock()
	return c
}

// Loading 加载数据
func (this *Cycle) Loading(name string) error {
	bytes, err := json.Marshal(newFile(name, "cycle").Get("data"))
	if err != nil {
		return err
	}
	data := []interface{}(nil)
	if err := json.Unmarshal(bytes, &data); err != nil {
		return err
	}
	for _, v := range data {
		if v == nil {
			this.Add(v)
		}
	}
	return nil
}

// Save 数据持久化,保存至文件
func (this *Cycle) Save(name string) error {
	return newFile(name, "cycle").Set("data", this.list, true)
}

// List 获取列表数据(时间正序)
func (this *Cycle) List(limit ...int) []interface{} {
	list := this.list[:this.offset]
	if this.cycle {
		list = append(this.list[this.offset:], list...)
	}
	if len(limit) > 0 && len(list) > limit[0] {
		return list[len(list)-limit[0]:]
	}
	return list
}

// Add 添加任意数据到缓存
func (this *Cycle) Add(i interface{}) *Cycle {
	data := i
	for _, v := range this.mChan {
		if v != nil {
			select {
			case v.C <- data:
			default:
			}
		}
	}
	this.list[this.offset] = data
	this.offset++
	if this.offset >= this.length {
		this.offset = 0
		this.cycle = true
	}
	return this
}

func newCycle(length int) *Cycle {
	return &Cycle{
		list:   make([]interface{}, length),
		offset: 0,
		length: length,
	}
}
