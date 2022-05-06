package splitter

import (
	"context"
	"fmt"
)

func ExampleSplitter_simple() {
	ctx := context.Background()

	square := func(x int) int {
		return x * x
	}

	// create a splitter, passing in a function and how many routines processing
	// jobs using this function you want
	sf := New[int, int](ctx, WithFunction(square, 5))
	for i := 0; i < 25; i++ {
		// add each job
		sf.Do(i)
	}
	// notify splitter that no more jobs are coming in
	sf.Done()

	// range over results works best because sf.Results() is closed when all
	// jobs have been processed
	for x := range sf.Results() {
		fmt.Println(x)
	}
}
