# split-for
[![Go Reference](https://pkg.go.dev/badge/badge/github.com/tannerhat/split-for.svg)](https://pkg.go.dev/github.com/tannerhat/split-for)
A little package for handling all the boilerplate stuff for splitting jobs amongst a pool of workers.
## As a Function
The actual logic backing anything in the package. Internally this function spins up goroutines for each worker func it is given and has them process jobs from a passed in channel until that channel closes, putting results into a results channel. It also handles premature stopping due to context cancellation, user cancellation, or error return from one of the worker funcs. As long as the user closes the jobs channel (or one of the stopping events happens), all goroutines will be cleaned up.

The function 
## As a Struct
