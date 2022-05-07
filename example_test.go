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
	jobChan := make(chan int, 25)
	sf := New[int, int](ctx, jobChan, WithFunction(square, 5))
	for i := 0; i < 25; i++ {
		// add each job
		jobChan <- i
	}
	// notify splitter that no more jobs are coming in
	close(jobChan)

	// range over results works best because sf.Results() is closed when all
	// jobs have been processed
	for x := range sf.Results() {
		fmt.Println(x)
	}
}
