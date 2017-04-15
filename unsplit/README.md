unsplit is for when the cost of creating a `[][]byte` from `bytes.Split` is a problem.
it successively returns a slice of each token in the string without creating a new [][]byte.

It can be 3 times faster than `bytes.Split()` and 2 times fater than `bytes.SplitN` (though both
of those functions are faster in go1.9+ so the improvement is less.

```
BenchmarkUnsplit-4   	10000000	       165 ns/op
BenchmarkSplit-4     	 3000000	       487 ns/op
BenchmarkSplitN-4    	 5000000	       292 ns/op
```
