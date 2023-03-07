package cache

import (
	"bytes"
	"fmt"
	"github.com/injoyai/base/maps"
	"github.com/injoyai/conv"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"time"
)

/*
NewFileLog
文件日志 例如接口请求日志
提供日志文件存储,历史日志查询,历史日志删除,曲线图

数据统计支持按秒(最小)



*/
func NewFileLog(cfg *FileLogConfig) *FileLog {
	//初始化
	f := new(FileLog)
	f.FileLogConfig = cfg.deal()
	//加载最新数据
	f.cacheLast = NewCycle(f.CacheNum)
	f.cacheFile = maps.NewSafe()
	//打开文件/创建文件
	var err error
	_ = os.MkdirAll(f.Dir, 0666)
	currentFilename := time.Now().Format(f.Layout)
	f.currentFile, err = os.OpenFile(filepath.Join(f.Dir, currentFilename), os.O_RDWR|os.O_APPEND, 0)
	if err == nil {
		f.currentFilename = currentFilename
	}
	return f
}

const (
	defaultFileLogSaveTime = time.Hour * 24 * 10
	defaultFileLogDir      = "./output/log/"
	defaultFileLogLayout   = "2006-01-02-15.log"
	defaultFileLogSplit    = "\n"
	defaultFileLogPerm     = 0666
)

// FileLogConfig 可选配置信息
type FileLogConfig struct {
	SaveTime         time.Duration //保存时长,根据写入新文件时触发(不写入不触发,文件颗粒度约细,触发越频繁)
	Dir              string        //文件保存目录 默认 "./output/log/"
	Layout           string        //文件命名规则 例如 "日志2006-01-02-15.log"
	CacheNum         int           //缓存最新数据大小
	CacheFileMaxSize int           //缓存文件大小(最大值),占用内存大小(当文件大小大于剩余大小则不缓存) 默认不占用
	Split            string        //文件内容分隔符 默认 "\n"
}

// deal 整理配置信息,未设置增加默认值
func (this *FileLogConfig) deal() *FileLogConfig {
	if this.SaveTime <= 0 {
		this.SaveTime = defaultFileLogSaveTime
	}
	if len(this.Dir) == 0 {
		this.Dir = defaultFileLogDir
	}
	if len(this.Layout) == 0 {
		this.Layout = defaultFileLogLayout
	}
	if this.CacheNum == 0 {
		this.CacheNum = 100
	}
	if len(this.Split) == 0 {
		this.Split = defaultFileLogSplit
	}
	os.MkdirAll(this.Dir, defaultFileLogPerm)
	return this
}

/*
FileLog 文件日志
单文件万次写入速度 0.08s/万次
单文件百万次写入速度 2.11s/百万次
多文件千万次写入速度 19.88/千万次
*/
type FileLog struct {
	*FileLogConfig

	cacheLast       *Cycle     //最新实时数据
	cacheFile       *maps.Safe //缓存读取过的文件信息
	cacheFileSize   int        //当前缓存文件大小
	currentFile     *os.File   //当前文件
	currentFilename string     //当前文件名称

}

// WriteAny 写入任意数据,根据配置写入到不同的文件
func (this *FileLog) WriteAny(p interface{}) (int, error) {
	return this.Write([]byte(conv.String(p)))
}

// Write 写入数据,根据配置写入到不同的文件,数据默认按'\n'分割
func (this *FileLog) Write(p []byte) (n int, err error) {

	now := time.Now()

	layout := now.Format(this.Layout)
	currentDir := filepath.Dir(filepath.Join(this.Dir, layout))
	currentFilename := filepath.Base(layout)

	if this.currentFile == nil {
		this.currentFile, err = os.OpenFile(filepath.Join(currentDir, currentFilename), os.O_RDWR|os.O_APPEND, defaultFileLogPerm)
		if err == nil {
			this.currentFilename = currentFilename
		}
	}

	if this.currentFile == nil || this.currentFilename != currentFilename {

		if this.currentFile != nil {
			//关闭老文件
			_ = this.currentFile.Close()
		}

		this.currentFilename = currentFilename

		//删除旧数据
		fs, _ := ioutil.ReadDir(currentDir)
		for _, v := range fs {
			if this.SaveTime > 0 && v.Name() < now.Add(-this.SaveTime).Format(this.Layout) {
				//清除缓存
				this.cacheFile.Del(v.Name())
				//移除文件
				os.Remove(this.Dir + v.Name())
			}
		}

		//新建日志文件
		os.MkdirAll(currentDir, defaultFileLogPerm)
		this.currentFile, err = os.Create(filepath.Join(currentDir, currentFilename))
		if err != nil {
			return 0, err
		}
	}

	if this.currentFile != nil && len(p) > 0 {
		n, err = this.currentFile.Write(append(p, []byte(this.Split)...))
		if err == nil {
			this.cacheLast.Add(p)
		}
	}

	return
}

// GetLogLast 读取文件最新数据
func (this *FileLog) GetLogLast(pageSize int) (list [][]byte, err error) {
	//查询缓存
	if data := this.cacheLast.List(); len(data) >= pageSize {
		for _, v := range data[:pageSize] {
			list = append(list, v.([]byte))
		}
		return
	}
	//数据不够则查询文件
	fs, err := this.rangeDir(this.Dir, nil)
	if err != nil {
		return nil, err
	}
	length := 0
	for i := len(fs) - 1; i >= 0; i-- {
		data, err := this.readFile(fs[i])
		if err != nil {
			return nil, err
		}
		for k := len(data) - 1; k >= 0; k-- {
			list = append(list, data[k])
			length++
			if length >= pageSize {
				return list, nil
			}
		}
	}
	return
}

// GetLog 读取文件数据
func (this *FileLog) GetLog(start, end time.Time) ([][]byte, error) {
	fs, err := this.rangeDir(this.Dir, nil)
	if err != nil {
		return nil, err
	}
	startFilename := start.Format(filepath.Join(this.Dir, this.Layout))
	endFilename := end.Format(filepath.Join(this.Dir, this.Layout))
	list := [][]byte(nil)
	for _, filepath := range fs {
		if filepath >= startFilename && filepath <= endFilename {
			data, err := this.readFile(filepath)
			if err != nil {
				return nil, err
			}
			filesize := len(data)
			if filesize == 0 {
				continue
			}
			//当文件大小小于剩余缓存空间大小,则加入缓存文件
			if this.cacheFileSize+filesize <= this.CacheFileMaxSize && filepath != this.currentFilename {
				this.cacheFile.Set(filepath, data)
				this.cacheFileSize += filesize
			}
			list = append(list, data...)
		}
	}
	return list, nil
}

// GetLogMerge 获取合并的数据,间隔至少1秒
func (this *FileLog) GetLogMerge(start, end time.Time, merge time.Duration, fn func([]byte) (IFileLogAny, error)) (map[int64][]IFileLogAny, error) {
	//统计间隔
	interval := int64(merge / time.Second)
	if interval <= 0 {
		return nil, fmt.Errorf("合并间隔有误(%s),至少需要1秒", merge)
	}
	//获取日志列表
	list, err := this.GetLog(start, end)
	if err != nil {
		return nil, err
	}
	//填充数据
	m := make(map[int64][]IFileLogAny)
	for i := start.Unix(); i <= end.Unix(); i += interval {
		m[i] = []IFileLogAny(nil)
	}
	//合并数据
	for _, v := range list {
		any, err := fn(v)
		if err != nil {
			return nil, err
		}
		if sec := any.GetSecond(); sec >= start.Unix() && sec <= end.Unix() {
			node := sec - (sec-start.Unix())%interval
			m[node] = append(m[node], any)
		}
	}
	return m, nil
}

// GetLogCurve 获取日志并生成曲线图
func (this *FileLog) GetLogCurve(start, end time.Time, merge time.Duration, i IFileLog) ([]interface{}, error) {
	m, err := this.GetLogMerge(start, end, merge, i.Decode)
	if err != nil {
		return nil, err
	}
	list := []interface{}(nil)
	for node, v := range m {
		data, err := i.Report(node, v)
		if err != nil {
			return nil, err
		}
		list = append(list, data)
	}
	sort.Sort(&_sort{list: list, compare: i.Compare})
	return list, nil
}

type IFileLogAny interface {
	GetSecond() int64
}

type IFileLog interface {
	Decode([]byte) (IFileLogAny, error)                         //字节转对象
	Report(node int64, list []IFileLogAny) (interface{}, error) //整理对象,合并统计
	Compare(a, b interface{}) bool                              //排序对象
}

func (this *FileLog) rangeDir(dir string, is func(string) bool) ([]string, error) {
	fileInfoList, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	list := []string(nil)
	for _, v := range fileInfoList {
		if v.IsDir() && (is == nil || is(v.Name())) {
			l, err := this.rangeDir(filepath.Join(dir, v.Name()), is)
			if err != nil {
				return nil, err
			}
			list = append(list, l...)
		} else {
			list = append(list, filepath.Join(dir, v.Name()))
		}
	}
	return list, nil
}

// readFile 读取文件数据,并分割整理数据
func (this *FileLog) readFile(filepath string) ([][]byte, error) {
	//判断缓存是否存在,存在则读取缓存的数据
	if val := this.cacheFile.MustGet(filepath); val != nil {
		return val.([][]byte), nil
	}
	f, err := os.Open(filepath)
	if err != nil {
		return nil, nil
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	list := bytes.Split(data, []byte(this.Split))
	if len(list) == 0 {
		return nil, nil
	}
	return list[:len(list)-1], nil
}

type _sort struct {
	list    []interface{}
	compare func(a, b interface{}) bool
}

func (this *_sort) Len() int {
	return len(this.list)
}

func (this *_sort) Less(i, j int) bool {
	if this.compare != nil {
		return this.compare(i, j)
	}
	return true
}

func (this *_sort) Swap(i, j int) {
	this.list[i], this.list[j] = this.list[j], this.list[i]
}
