# split-for
[![Go Reference](https://pkg.go.dev/badge/badge/github.com/tannerhat/split-for.svg)](https://pkg.go.dev/github.com/tannerhat/split-for)
A little package for handling all the boilerplate stuff for splitting jobs amongst a pool of workers. There's a few ways to use based on how you want to interact with the workers.
## As a Function
### Split[J any, R any]
The actual logic backing everything else. Internally this function spins up goroutines for each worker func it is given and has them process jobs from a passed in channel until that channel closes. It puts results into a results channelc. It also handles premature stopping due to context cancellation, user cancellation, or error return from one of the worker funcs. As long as the user closes the jobs channel (or one of the stopping events happens), all goroutines will be cleaned up.
```
square := func(x int) (int, error) {
    return x * x, nil
}

jobs := make(chan int, 100)
funcs := []func(int) (int, error){square, square, square}
// split the jobs among the funcs (do not exit on func failure)
results, _, _ := Split[int, int](ctx, jobs, funcs, true)
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
```
### SplitSlice[J any, R any]
Send a slice of jobs and get back a slice of results in the same order. This turns Split into
a synchronous function, but simplifies the interface.
```
square := func(x int) int {
    return x * x
}

ctx := context.Background()
jobs := []int{}
for i := 0; i < 100; i++ {
    jobs = append(jobs, i)
}

results, err := SplitSlice(ctx, jobs, square, 100)

for x := range results {
    fmt.Println(x)
}
```
## As a Struct
### Splitter[J any, R any]
A struct wrapped around a call to Split[J,R]. Replaces the jobs channel with methods `Do` and `Done` and gives a `Close()` method.
```
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
```
