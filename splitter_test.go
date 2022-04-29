package splitter

import (
	"context"
	"reflect"
	"sort"
	"testing"
	"time"
)

func square(x int) int {
	return x * x
}

func TestSplitter(t *testing.T) {
	ctx := context.Background()

	exp := make([]int, 25)
	ret := []int{}
	sf := New[int, int](ctx, WithFunction(square, 5))
	for i := 0; i < 25; i++ {
		sf.Do(i)
		exp[i] = i * i
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
	if !reflect.DeepEqual(ret, exp) {
		t.Errorf("bad returns. got:%v wanted:%v", ret, exp)
	}
}

func TestSplitterWithFuncs(t *testing.T) {
	ctx := context.Background()

	// let's validate that if we pass multiple funcs through the WithFunctions option
	// they all get called
	numFuncs := 5
	numJobs := 100
	called := make([]bool, numFuncs)
	exp := make([]int, numJobs)
	ret := []int{}
	funcs := []func(int) int{}
	// make funcs and have them each set their index in called
	for i := 0; i < numFuncs; i++ {
		// do this in a func to properly capture i so we can mark called
		func(idx int) {
			funcs = append(funcs, func(val int) int {
				// sleep a moment to make sure the other workers pick up too
				time.Sleep(time.Millisecond * 10)
				called[idx] = true
				return square(val)
			})
		}(i)
	}

	// create splitter with the funcs
	sf := New[int, int](ctx, WithFunctions[int, int](funcs))

	// send all the jobs
	for i := 0; i < numJobs; i++ {
		sf.Do(i)
		exp[i] = i * i
	}
	// Done should have the splitter close the output channel once all jobs are done
	sf.Done()

	// read results in a goroutine in case it hangs
	done := make(chan bool)
	go func() {
		for x := range sf.Results() {
			ret = append(ret, x)
		}
		close(done)
	}()

	// wait for either it finished or timeout
	select {
	case <-done:
		break
	case <-time.After(time.Second):
		t.Fatalf("test timed out, splitter might be hung")
	}

	// sort the two because the jobs may complete out of order
	sort.Ints(ret)
	sort.Ints(exp)
	if !reflect.DeepEqual(ret, exp) {
		t.Errorf("bad returns. got:%v wanted:%v", ret, exp)
	}

	// all the workers should be called, it's not really guaranteed, but really should happen
	// with a large enough job set.
	for i, c := range called {
		if !c {
			t.Errorf("function %d not called. called funcs:%v", i, called)
		}
	}
}
