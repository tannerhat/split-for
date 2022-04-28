package splitter

import (
	"context"
	"testing"
	"time"
	"sort"
	"reflect"
)

func square(x int) int {
	return x * x
}

func TestSplitter(t *testing.T) {
	ctx:=context.Background()

	exp := make([]int, 25)
	ret := []int{}
	sf := New[int, int](ctx, square, 5)
	for i := 0; i < 25; i++ {
		sf.Do(i)
		exp[i]=i*i
	}
	sf.Done()

	done := make(chan bool)
	go func() {
		for x := range sf.Results() {
			ret = append(ret, x)
		}
		close(done)
	}()

	select {
	case <-done:
		break
	case <-time.After(time.Second):
		t.Errorf("test timed out, splitter might be hung")
	}

	sort.Ints(ret)
	sort.Ints(exp)
	if !reflect.DeepEqual(ret,exp) {
		t.Errorf("bad returns. got:%v wanted:%v",ret,exp)
	}
}