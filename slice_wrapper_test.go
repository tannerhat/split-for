package splitter

import (
	"context"
	"reflect"
	"testing"
)

func TestSliceSplitter(t *testing.T) {
	ctx := context.Background()
	jobs := []int{}
	exp := []int{}
	for i := 0; i < 100; i++ {
		jobs = append(jobs, i)
		exp = append(exp, i*i)
	}

	results, err := SplitSplice(ctx, jobs, square, 100)
	if err != nil {
		t.Errorf("unexpected splitter failure:%s", err)
	}
	if !reflect.DeepEqual(exp, results) {
		t.Errorf("bad results returned. got:%v wanted:%v", results, exp)
	}
}
