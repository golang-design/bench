# bench

Reliable performance measurement for Go programs. All in one design.

```
$ go install golang.design/x/bench@latest
```

## Features

- Combine [benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat), [perflock](https://github.com/aclements/perflock) and more...
- Short command and only run benchmarks
- Automatic performance locking for benchmarks
- Automatic statistic analysis for benchmark results
- Color indications for benchmark results

## Usage

### Enable `bench` Daemon (optional, Linux only)

```sh
$ cd $GOPATH/src/golang.design/x/bench
$ ./install.bash
```

If your init system is supported, this will also configure `bench` to start automatically on boot.

Or you can install and run `bench` daemon manually:

```sh
$ sudo install $GOPATH/bin/bench /usr/bin/bench
$ sudo -b bench -daemon
```

### Default Behavior

```sh
$ bench
```

The detault behavior of `bench` run benchmarks under your current
working directory, and each benchmark will be ran 10 times for further
statistical analysis. It will also try to acquire performance lock from
`bench` daemon to gain more stable results. Furthermore, the benchmark
results are saved as a text file to the working directory and named as
`<timestamp>.txt`.

Example:

```
$ cd example
$ bench
bench: run benchmarks under 90% cpufreq...
bench: go test -run=^$ -bench=. -count=10
goos: linux
goarch: amd64
pkg: golang.design/x/bench/example
BenchmarkDemo-16           21114             57340 ns/op
...
BenchmarkDemo-16           21004             57097 ns/op
PASS
ok      golang.design/x/bench/example   17.791s
bench: results are saved to file: ./bench-2020-11-07-19:59:51.txt

name     time/op
Demo-16  57.0µs ±1%

$ # ... do some changes to the benchmark ...

$ bench
bench: run benchmarks under 90% cpufreq...
bench: go test -run=^$ -bench=. -count=10
goos: linux
goarch: amd64
pkg: golang.design/x/bench/example
BenchmarkDemo-16          213145              5625 ns/op
...
BenchmarkDemo-16          212959              5632 ns/op
PASS
ok      golang.design/x/bench/example   12.536s
bench: results are saved to file: ./bench-2020-11-07-20:00:16.txt

name     time/op
Demo-16  5.63µs ±0%

$ bench bench-2020-11-07-19:59:51.txt bench-2020-11-07-20:00:16.txt
name     old time/op new time/op  delta
Demo-16  57.0µs ±1%  5.6µs ±0%   -90.13%  (p=0.000 n=10+8)
```

### Options

Options for checking daemon status:

```sh
bench -list
```

Options for statistic tests:

```sh
bench old.txt [new.txt]             # same from benchstat
bench -delta-test
bench -alpha
bench -geomean
bench -split
bench -sort
```

Options for running benchmarks:

```sh
bench -v                            # enable verbose outputs
bench -shared                       # enable shared execution
bench -cpufreq 90                   # cpu frequency             (default: 90)
bench -name BenchmarkXXX            # go test `-bench` flag     (default: .)
bench -count 20                     # go test `-count` flag     (default: 10)
bench -time 100x                    # go test `-benchtime` flag (default: unset)
bench -cpuproc 1,2,4,8,16,32,128    # go test `-cpu` flag       (default: unset)
```

## License

&copy; 2020 The golang.design Authors