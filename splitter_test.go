package splitter

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"testing"
	"time"
)

func square(x int) int {
	fmt.Println(x * x)
	return x * x
}

func squareError(x int) (int, error) {
	if x == -1 {
		// to test error causing cancel
		return 0, errTestCancelError
	}
	return x * x, nil
}

func TestSplitter(t *testing.T) {
	ctx := context.Background()

	exp := make([]int, 25)
	ret := []int{}
	jobChan := make(chan int, 25)
	sf := New[int, int](ctx, jobChan, WithFunction(square, 5))
	for i := 0; i < 25; i++ {
		jobChan <- i
		exp[i] = i * i
	}
	close(jobChan)

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

func TestCancel(t *testing.T) {
	cancelTest(t, userCancel)
	cancelTest(t, contextCancel)
	cancelTest(t, errorCancel)
}

type cancelType string

const (
	userCancel    cancelType = "user"
	contextCancel cancelType = "context"
	errorCancel   cancelType = "error"
)

var errTestCancelError = fmt.Errorf("test cancel")

func cancelTest(t *testing.T, reason cancelType) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())

	// have a splitter run a bunch of jobs
	cancelTime := 1000
	readDone := make(chan bool)

	jobChan := make(chan int, 1000)
	sf := New[int, int](ctx, jobChan, WithErrorFunction(squareError, 5), StopOnError[int, int]())
	// in a routine, add jobs and do a cancel, it's in a routine so we can check that
	// the splitter exits right away while we keep adding jobs
	stopAdd := make(chan bool)
	go func() {
		defer func() {
			close(jobChan)
		}()

		for i := 0; ; i++ {
			jobChan <- i
			if i == cancelTime {
				// at some point, cause the cancel. We will keep adding
				// jobs though just to make it more difficult for splitter
				if reason == contextCancel {
					go cancel()
				} else if reason == userCancel {
					sf.Cancel()
				} else if reason == errorCancel {
					jobChan <- -1
				}
			}
			select {
			case <-stopAdd:
				// ok we've got the signal that the splitter is done. we should leave now.
				return
			default:
				// keep on addin'
			}
		}
	}()

	// in a routine read jobs, in a routine so we can validate that first we
	// get an error saying we cancelled and then the results get closed
	go func() {
		for range sf.Results() {
		}
		close(readDone)
	}()

	// we should get the error first
	select {
	case err := <-sf.Errors():
		if reason == contextCancel && !errors.Is(err, ErrContextCancel) {
			t.Errorf("wrong error returned by splitter. wanted:%s got:%s", ErrContextCancel, err)
		} else if reason == userCancel && !errors.Is(err, ErrUserCancel) {
			t.Errorf("wrong error returned by splitter. wanted:%s got:%s", ErrUserCancel, err)
		} else if reason == errorCancel && !errors.Is(err, errTestCancelError) {
			t.Errorf("wrong error returned by splitter. wanted:%s got:%s", errTestCancelError, err)
		}
	case <-readDone:
		t.Errorf("expected error first, but read finished first")
	case <-time.After(time.Second):
		t.Fatalf("expected error first, but test timed out")
	}

	// next the reads should close
	select {
	case <-readDone:
		break
	case <-time.After(time.Second):
		t.Fatalf("expected read finish next, but test timed out")
	}

	// we can stop adding now
	close(stopAdd)
}

func TestErrorReturn(t *testing.T) {
	ctx := context.Background()

	exp := []int{}
	ret := []int{}
	jobChan := make(chan int, 1000)
	sf := New[int, int](ctx, jobChan, WithErrorFunction(squareError, 20))

	stopAdd := make(chan bool)
	go func() {
		for i := 0; ; i++ {
			select {
			case <-stopAdd:
				return
			default:
			}
			jobChan <- i
			// it can fail if the channel fills up, just ignore it
			// but don't add to expected
			exp = append(exp, i*i)
			time.Sleep(time.Millisecond * 10)
		}
	}()

	readDone := make(chan bool)
	go func() {
		for x := range sf.Results() {
			ret = append(ret, x)
		}
		close(readDone)
	}()

	// run for a bit to get some processing done
	time.Sleep(time.Millisecond * 50)

	// we shouldn't have any errors yet
	select {
	case err := <-sf.Errors():
		t.Errorf("unexpected error:%s", err)
	default:
	}

	// let's cause a few errors
	for i := 0; i < 3; i++ {
		// cause an error
		jobChan <- -1
		select {
		case err := <-sf.Errors():
			if !errors.Is(err, errTestCancelError) {
				t.Errorf("unexpected error.got:%s wanted:%s", err, errTestCancelError)
			}
		case <-time.After(time.Second):
			t.Fatalf("never got our error")
		}

		// the error shouldn't have stopped proccessing, confirm by checking length, waiting
		// and checking again. it should grow
		startLen := len(ret)
		time.Sleep(time.Millisecond * 50)
		if len(ret) == startLen {
			t.Errorf("ret isn't growing, splitter stopped after error")
		}
	}

	// now let's finish and compare results
	close(stopAdd)
	close(jobChan)
	select {
	case <-readDone:
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
	jobChan := make(chan int, 1000)
	sf := New[int, int](ctx, jobChan, WithFunctions[int, int](funcs))

	// send all the jobs
	for i := 0; i < numJobs; i++ {
		jobChan <- i
		exp[i] = i * i
	}
	// Done should have the splitter close the output channel once all jobs are done
	close(jobChan)

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
