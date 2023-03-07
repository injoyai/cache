package cache

import (
	"testing"
	"time"
)

// TestNewFileLogWrite 测试写入速度
func TestNewFileLogWrite(t *testing.T) {
	f := NewFileLog(&FileLogConfig{
		Layout: "2006-01/02-15-04-05.log"})
	for i := 0; i < 1000*10000; i++ {
		f.WriteAny(i)
	}

	t.Log(f.GetLog(time.Time{}, time.Now()))
	t.Log(f.GetLog(time.Time{}, time.Now()))
}

func TestNewFileLog(t *testing.T) {
	f := NewFileLog(&FileLogConfig{})

	go func() {
		for i := 0; i < 6000; i++ {
			<-time.After(time.Millisecond * 1)
			f.WriteAny(i)
		}
	}()

	<-time.After(time.Second * 2)
	result, err := f.GetLog(time.Time{}, time.Now())
	if err != nil {
		t.Error(err)
		return
	}

	for _, v := range result {
		t.Log(string(v))
	}
}

// TestNewFileLogGet 测试文件读取速度
func TestNewFileLogGet(t *testing.T) {
	f := NewFileLog(&FileLogConfig{
		Layout:           "2006-01/02-15-04-05.log",
		CacheFileMaxSize: 10 << 20,
	})

	data, err := f.GetLog(time.Time{}, time.Now())
	if err != nil {
		t.Error(err)
	}
	t.Log(len(data))
	data, err = f.GetLog(time.Time{}, time.Now())
	if err != nil {
		t.Error(err)
	}
	t.Log(len(data))
}
