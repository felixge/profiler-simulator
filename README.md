# profiler-simulator

profiler-simulator is a program that simulates the results different profiler
implementations will give for the execution of a simulated program. Currently
only single threaded execution is implemented, but that should be good enough to
simulate many interesting scenarios.

Below is the example output for a 100s run of a simulated program that looks
like this:

```go
func main() {
    for {
        workA() // 13ms
        workB() // 25ms
        sleep() // 62ms
    }
}
```

```
$ go run .
Perfect Wallclock Profiler
1m2s: (off-cpu) main;sleep
25s: (on-cpu) main;workB
13s: (on-cpu) main;workA

Perfect CPU Profiler
25s: (on-cpu) main;workB
13s: (on-cpu) main;workA

Node.js Wallclock Profiler
25.252525s: (on-cpu) main;workB
12.121212s: (on-cpu) main;workA

Go CPU Profiler
26s: (on-cpu) main;workB
12s: (on-cpu) main;workA
```