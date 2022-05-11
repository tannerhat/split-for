package splitter

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestSliceSplitter(t *testing.T) {
	ctx := context.Background()
	jobs := []int{}
	exp := []int{}
	for i := 0; i < 100; i++ {
		jobs = append(jobs, i)
		exp = append(exp, i*i)
	}

	results, err := SplitSlice(ctx, jobs, square, 100)
	if err != nil {
		t.Errorf("unexpected splitter failure:%s", err)
	}
	if !reflect.DeepEqual(exp, results) {
		t.Errorf("bad results returned. got:%v wanted:%v", results, exp)
	}
}

func stall(i int) int {
	time.Sleep(time.Second * 5)
	return i
}

func TestSliceSplitterTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	jobs := []int{}
	for i := 0; i < 100; i++ {
		jobs = append(jobs, i)
	}

	_, err := SplitSlice(ctx, jobs, stall, 100)
	if err == nil {
		t.Errorf("expected splitter to context timeout")
	} else if !errors.Is(err, ErrContextCancel) {
		t.Errorf("wrong error returned. got:%s wanted:%s", err, ErrContextCancel)
	}
}
