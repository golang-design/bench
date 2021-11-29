// Copyright 2020 The golang.design Initiative Authors.
// All rights reserved. Use of this source code is governed
// by a GNU GPLv3 license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.design/x/bench/internal/lock"
	"golang.design/x/bench/internal/stat"
	"golang.design/x/bench/internal/term"
)

func usage() {
	fmt.Fprintf(os.Stderr, `usage: bench [options]
options for daemon usage:
	-daemon
		run bench service
	-list
		print current and pending commands

options for significant tests:
	-delta-test test
		significance test to apply to delta: utest, ttest, or none (default "utest")
	-alpha α
		consider change significant if p < α (default 0.05)
	-geomean
		print the geometric mean of each file (default false)
	-split labels
		split benchmarks by labels (default "pkg,goos,goarch")
	-sort order
		sort by order: [-]delta, [-]name, none (default "none")

options for running benchmarks:
	-v go test
		the -v flag from go test, (default false)
	-name go test
		the -bench flag from go test (default .) (default ".")
	-count go test
		the -count flag from go test (default 10) (default 10)
	-time go test
		the -benchtime flag from go test (default unset)
	-cpuprocs go test
		the -cpu flag to go test (default unset)

options for performance locking
	-shared
		acquire lock in shared mode (default exclusive mode)
	-cpufreq percent
		set CPU frequency to percent between the min and max (default 90)
		while running command, or "none" for no adjustment
`)
	os.Exit(2)
}

var (
	flagDaemon *bool
	flagList   *bool

	flagDeltaTest *string
	flagAlpha     *float64
	flagGeomean   *bool
	flagSplit     *string
	flagSort      *string

	flagShared  *bool
	flagCPUFreq *lock.CpufreqFlag

	flagVerbose  *bool
	flagName     *string
	flagCount    *int
	flagTime     *string
	flagCPUProcs *string
)

func main() {
	log.SetPrefix("bench: ")
	log.SetFlags(0)
	flag.Usage = usage

	// daemon args
	flagDaemon = flag.Bool("daemon", false, "run bench service")
	flagList = flag.Bool("list", false, "print current and pending commands")

	// benchstat args
	flagDeltaTest = flag.String("delta-test", "utest", "significance `test` to apply to delta: utest, ttest, or none")
	flagAlpha = flag.Float64("alpha", 0.05, "consider change significant if p < `α`")
	flagGeomean = flag.Bool("geomean", false, "print the geometric mean of each file")
	flagSplit = flag.String("split", "pkg,goos,goarch", "split benchmarks by `labels`")
	flagSort = flag.String("sort", "none", "sort by `order`: [-]delta, [-]name, none")

	// perflock flags
	flagShared = flag.Bool("shared", false, "acquire lock in shared mode (default exclusive mode)")
	flagCPUFreq = &lock.CpufreqFlag{Percent: 90}
	flag.Var(flagCPUFreq, "cpufreq", "set CPU frequency to `percent` between the min and max\n\twhile running command, or \"none\" for no adjustment")

	// go test args
	flagVerbose = flag.Bool("v", false, "the -v flag from `go test`, (default false)")
	flagName = flag.String("name", ".", "the -bench flag from `go test` (default .)")
	flagCount = flag.Int("count", 10, "the -count flag from `go test` (default 10)")
	flagTime = flag.String("time", "", "the -benchtime flag from `go test` (default unset)")
	flagCPUProcs = flag.String("cpuprocs", "", "the -cpu flag to `go test` (default unset)")
	flag.Parse()

	if *flagDaemon {
		if flag.NArg() > 0 {
			flag.Usage()
			os.Exit(2)
		}
		lock.RunDaemon()
		return
	}
	if *flagList {
		if flag.NArg() > 0 {
			flag.Usage()
			os.Exit(2)
		}
		c := lock.NewClient()
		if c == nil {
			log.Fatal("Is the bench daemon running?")
		}
		list := c.List()
		if len(list) == 0 {
			log.Println("daemon is running but no running benchmarks.")
			return
		}
		for _, l := range list {
			log.Println(l)
		}
		return
	}

	if flag.NArg() > 0 {
		runCompare()
		return
	}

	// prepare go test command
	args := []string{
		"go",
		"test",
		"-run=^$",
	}
	if *flagVerbose {
		args = append(args, "-v")
	}
	args = append(args, fmt.Sprintf("-bench=%s", *flagName))
	if *flagCount <= 0 {
		*flagCount = 10
	}
	args = append(args, fmt.Sprintf("-count=%d", *flagCount))
	if *flagTime != "" {
		args = append(args, fmt.Sprintf("-benchtime=%s", *flagTime))
	}
	if *flagCPUProcs != "" {
		args = append(args, fmt.Sprintf("-cpu=%s", *flagCPUProcs))
	}

	// acquire lock
	c := lock.NewClient()
	if c == nil {
		log.Printf(term.Red("run benchmarks without performance locking..."))
	} else {
		if !c.Acquire(*flagShared, true, strings.Join(args, " ")) {
			list := c.List()
			log.Printf("Waiting for lock...\n")
			for _, l := range list {
				log.Println(l)
			}
			c.Acquire(*flagShared, false, strings.Join(args, " "))
		}
		if !*flagShared && flagCPUFreq.Percent >= 0 {
			c.SetCPUFreq(flagCPUFreq.Percent)
			log.Print(term.Gray(fmt.Sprintf("run benchmarks under %d%% cpufreq...", flagCPUFreq.Percent)))
		}
	}

	// Ignore SIGINT and SIGQUIT so they pass through to the
	// child.
	signal.Notify(make(chan os.Signal), os.Interrupt, syscall.SIGQUIT)

	// run bench
	runBench(args)
}

// shellEscape escapes a single shell token.
func shellEscape(x string) string {
	if len(x) == 0 {
		return "''"
	}
	for _, r := range x {
		if 'a' <= r && r <= 'z' || 'A' <= r && r <= 'Z' || '0' <= r && r <= '9' || strings.ContainsRune("@%_-+:,./", r) {
			continue
		}
		// Unsafe character.
		return "'" + strings.Replace(x, "'", "'\"'\"'", -1) + "'"
	}
	return x
}

// shellEscapeList escapes a list of shell tokens.
func shellEscapeList(xs []string) string {
	out := make([]string, len(xs))
	for i, x := range xs {
		out[i] = shellEscape(x)
	}
	return strings.Join(out, " ")
}

var deltaTestNames = map[string]stat.DeltaTest{
	"none":   stat.NoDeltaTest,
	"u":      stat.UTest,
	"u-test": stat.UTest,
	"utest":  stat.UTest,
	"t":      stat.TTest,
	"t-test": stat.TTest,
	"ttest":  stat.TTest,
}

func runCompare() {
	c := &stat.Collection{
		Alpha:      *flagAlpha,
		AddGeoMean: *flagGeomean,
		DeltaTest:  deltaTestNames[strings.ToLower(*flagDeltaTest)],
	}

	for _, file := range flag.Args() {
		f, err := os.Open(file)
		if err != nil {
			log.Print(err)
			flag.Usage()
			os.Exit(2)
		}
		defer f.Close()

		if err := c.AddFile(file, f); err != nil {
			log.Fatal(err)
		}
	}

	tables := c.Tables()
	var buf bytes.Buffer
	stat.FormatText(&buf, tables)
	os.Stdout.Write(buf.Bytes())
}

func runBench(args []string) {
	log.Print(strings.Join(args, " "))
	cmd := exec.Command(args[0], args[1:]...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}
	doneCh := make(chan []byte)
	go func() {
		data := []byte{}
		buf := make([]byte, 1024)
		for {
			n, err := stdout.Read(buf)
			if err != nil {
				if err != io.EOF {
					fmt.Fprintf(os.Stderr, "%v", err)
				}
				doneCh <- data
				close(doneCh)
				return
			}
			fmt.Fprintf(os.Stdout, string(buf[:n]))
			data = append(data, buf[:n]...)
		}
	}()
	errCh := make(chan error)
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stderr.Read(buf)
			if err != nil {
				if err == io.EOF {
					errCh <- nil
				} else {
					errCh <- err
				}
				close(errCh)
				return
			}
			fmt.Fprintf(os.Stderr, string(buf[:n]))
		}
	}()

	err = cmd.Start()
	switch err := err.(type) {
	case *exec.ExitError:
		status := err.Sys().(syscall.WaitStatus)
		if status.Exited() {
			os.Exit(status.ExitStatus())
		}
		log.Fatal(err)
	}

	var results []byte
	select {
	case err = <-errCh:
		if err != nil { // benchmark was interrupted or not success, exit.
			os.Exit(2)
			return
		}
	case results = <-doneCh:
	}

	// do nothing if no tests were ran.
	if strings.Index(string(results), "no Go files") != -1 ||
		strings.Index(string(results), "no test files") != -1 {
		return
	}

	// Note that we should avoid using : in filename, because it is not
	// supported on Windows file systems.
	fname := "bench-" + time.Now().Format("2006-01-02-15-04-05") + ".txt"
	err = ioutil.WriteFile(fname, results, 0644)
	if err != nil {
		// try again, maybe the user was too fast?
		err = ioutil.WriteFile(fname, results, 0644)
		if err != nil {
			log.Fatal("cannot save benchmark result to your disk.")
		}
	}
	log.Printf("results are saved to file: ./%s\n\n", fname)

	computeStat(results)
}

var sortNames = map[string]stat.Order{
	"none":  nil,
	"name":  stat.ByName,
	"delta": stat.ByDelta,
}

func computeStat(data []byte) {
	sortName := *flagSort
	reverse := false
	if strings.HasPrefix(sortName, "-") {
		reverse = true
		sortName = sortName[1:]
	}
	order, _ := sortNames[sortName]
	c := &stat.Collection{
		Alpha:      *flagAlpha,
		AddGeoMean: *flagGeomean,
		DeltaTest:  deltaTestNames[strings.ToLower(*flagDeltaTest)],
	}

	if *flagSplit != "" {
		c.SplitBy = strings.Split(*flagSplit, ",")
	}
	if order != nil {
		if reverse {
			order = stat.Reverse(order)
		}
		c.Order = order
	}
	c.AddData("", data)
	tables := c.Tables()
	var buf bytes.Buffer
	stat.FormatText(&buf, tables)
	fmt.Print(buf.String())
}
