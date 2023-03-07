package cache

import (
	"github.com/injoyai/base/chans"
	"github.com/injoyai/conv"
	"testing"
	"time"
)

func TestNewVar(t *testing.T) {

	//t.Log(NewFile().Sync().Set("test", 1))
	//t.Log(NewFile().Sync().Set("age", 18))
	//t.Log(NewFile().GetInt("test"))
	//t.Log(NewFile().GetInt("age"))
	//t.Log(NewFile().Sync().Set("age", 11))
	//t.Log(NewFile().GetInt("age"))
	//logs.PrintErr(New().Clear().Save())

}

//百万次单线程速度4.31秒
//百万次携程速度3.76秒
func TestNewCycle(t *testing.T) {

	x := NewCycle(1000)
	t.Log(x.Loading("666"))
	t.Log("x.List():", x.List())
	for _, v := range x.List() {
		t.Log(v)
	}

	for i := range chans.Count(1) {
		//go func(i int) {
		x.Add(i)
		x.List()
		//}(i)
	}
	t.Log(x.Save("666"))

}

func TestNewMap(t *testing.T) {
	x := NewMap()
	for i := range chans.Count(1000000) {
		//go func() {
		if i%2 == 0 {
			x.Set(conv.String(i), i, time.Second)
		} else {
			x.Set(conv.String(i), i)
		}
		x.GetInt(conv.String(i - 1))
		//}()
	}
}
