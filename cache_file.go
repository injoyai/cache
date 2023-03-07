package cache

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/injoyai/conv"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

type File struct {
	name string
	tag  string
	sync bool
	m    map[string]interface{}
	mu   sync.RWMutex
	conv.Extend
}

// GetVar 实现接口
func (this *File) GetVar(key string) *conv.Var {
	this.mu.RLock()
	defer this.mu.RUnlock()
	return conv.New(this.m[key])
}

func newFile(name, tag string) *File {
	data := new(File)
	data.name = name
	data.tag = tag
	bs, _ := ioutil.ReadFile(data.getPath(tag, name))
	m := make(map[string]interface{})
	_ = json.Unmarshal(bs, &m)
	data.m = m
	data.Extend = conv.NewExtend(data)
	return data
}

// Name 名字
func (this *File) Name() string {
	return this.name
}

// Clear 清空数据
func (this *File) Clear() *File {
	this.mu.Lock()
	defer this.mu.Unlock()
	this.m = make(map[string]interface{})
	return this
}

// Sync 同步覆盖更新到缓存,默认手动覆盖更新
func (this *File) Sync(b ...bool) *File {
	this.sync = conv.GetDefaultBool(true, b...)
	return this
}

// Set 设置参数
func (this *File) Set(key string, val interface{}, cover ...bool) error {
	this.mu.Lock()
	this.m[key] = val
	this.mu.Unlock()
	if this.sync || len(cover) > 0 && cover[0] {
		return this.Cover()
	}
	return nil
}

// Get 获取参数
func (this *File) Get(key string) *conv.Var {
	this.mu.RLock()
	defer this.mu.RUnlock()
	return conv.New(this.m[key])
}

// Del 删除参数
func (this *File) Del(key string, cover ...bool) error {
	this.mu.Lock()
	delete(this.m, key)
	this.mu.Unlock()
	if this.sync || len(cover) > 0 && cover[0] {
		return this.Cover()
	}
	return nil
}

// Cover 覆盖
func (this *File) Cover() error {
	path := this.getPath(this.tag, this.name)
	dir := filepath.Dir(path)
	name := filepath.Base(path)
	if err := os.MkdirAll(dir, 0666); err != nil {
		return err
	}
	if len(name) == 0 {
		return nil
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(conv.Bytes(this.m))
	return err
}

func (this *File) getPath(tag, name string) string {
	fileDir, filename := DefaultDir, name
	if dir, file := filepath.Split(name); len(dir) > 0 {
		fileDir, filename = dir, file
	}
	h := md5.New()
	h.Write([]byte(filename))
	return fileDir + fmt.Sprintf("%s@%s", tag, hex.EncodeToString(h.Sum(nil)))
}
