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
	sf := NewSplitter[int, int](ctx, FromFunction(square, 5))
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

	multFactory := func(m int) func(int) (int, error) {
		return func(x int) (int, error) { return x * m, nil }
	}

	jobs := make(chan int, 100)
	// passing 3 different functions, each gets a worker and will pull jobs and send results.
	funcs := []func(int) (int, error){multFactory(1), multFactory(2), multFactory(3)}
	// split the jobs among the funcs (exit if any fuction returns an error)
	results, errors, _ := Split[int, int](ctx, jobs, funcs, StopOnError())
	for i := 0; i < 25; i++ {
		// add each job, can be done before or after passing to Split
		jobs <- i
	}
	// notify splitter routines that no more jobs are coming in
	close(jobs)

	// range over results works best because results is closed when all
	// jobs have been processed
	for x := range results {
		fmt.Println(x)
	}

	select {
	case err := <-errors:
		fmt.Printf("it failed %s\n", err)
	default:
	}
}

func ExampleSplitSlice_simple() {
	square := func(x int) int {
		return x * x
	}

	ctx := context.Background()
	jobs := []int{}
	for i := 0; i < 100; i++ {
		jobs = append(jobs, i)
	}

	results, _ := SplitSlice(ctx, jobs, FromFunction(square, 100))

	for x := range results {
		fmt.Println(x)
	}
}
