package splitter

import (
	"context"
	"fmt"
	"time"
)

func ExampleSplitter_simple() {
	ctx := context.Background()

	square := func(x int) int {
		return x * x
	}

	// create a splitter, passing in a function and how many routines processing
	// jobs using this function you want
	sf := NewSplitter[int, int](ctx, WithFunction(square, 5))
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

func ExampleSplit_simple() {
	ctx := context.Background()

	square := func(x int) (int, error) {
		return x * x, nil
	}

	jobs := make(chan int, 100)
	funcs := []func(int) (int, error){square, square, square}
	// split the jobs among the funcs (do not exit on func failure)
	results, errors, cancel := Split[int, int](ctx, jobs, funcs, false)
	for i := 0; i < 25; i++ {
		// add each job, can be done before or after passing to Split
		jobs <- i
	}
	// notify splitter that no more jobs are coming in
	close(jobs)

	// routine to monitor errors and cancel if needed
	for {
		select {
		case err, ok := <-errors:
			if !ok {
				// channel close, leave routine
				return
			}
			// print the error
			fmt.Printf("failure: %s\n", err)
		case <-time.After(time.Second):
			cancel()
		}
	}

	// range over results works best because sf.Results() is closed when all
	// jobs have been processed
	for x := range results {
		fmt.Println(x)
	}
}
