package main

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"
)

func main() {
	program := Repeat(100*time.Second, []*StackTrace{
		{Duration: 13 * time.Millisecond, Frames: []string{"main", "workA"}, CPU: true},
		{Duration: 25 * time.Millisecond, Frames: []string{"main", "workB"}, CPU: true},
		{Duration: 62 * time.Millisecond, Frames: []string{"main", "sleep"}},
	})

	for _, p := range profilers {
		profile := p.Profile(program)
		profile.Sort()
		fmt.Printf("%s\n%s\n", p.Name, profile)
	}
}

type Profiler struct {
	Name    string
	Profile func(e *ExecutionTrace) *Profile
}

var profilers = []*Profiler{
	{
		Name: "Perfect Wallclock Profiler",
		Profile: func(e *ExecutionTrace) *Profile {
			var p Profile
			for _, sample := range e.Calls {
				p.Add(sample)
			}
			return &p
		},
	},
	{
		Name: "Perfect CPU Profiler",
		Profile: func(e *ExecutionTrace) *Profile {
			var p Profile
			for _, sample := range e.Calls {
				if !sample.CPU {
					continue
				}
				p.Add(sample)
			}
			return &p
		},
	},
	{
		Name: "Node.js Wallclock Profiler",
		Profile: func(e *ExecutionTrace) *Profile {
			const hz = 99
			var p Profile
			period := time.Second / hz
			var offset time.Duration
			for {
				offset += period
				if sample := e.Sample(offset); sample == nil {
					return &p // end of profile
				} else if sample.CPU {
					sample = sample.Copy()
					sample.Duration = period
					p.Add(sample)
				}
			}
		},
	},
	{
		Name: "Go CPU Profiler",
		Profile: func(e *ExecutionTrace) *Profile {
			// The code below attempts to simulates how the kernel uses its
			// ijffy interval (4ms) for cpu-time accounting and signal
			// generation when using POSIX timer APIs like setitimer(2),
			// timer_create(2).
			// https://felixge.de/2022/02/11/profiling-improvements-in-go-1.18/
			var (
				p           Profile
				cpuDuration time.Duration
				jiffy       = 4 * time.Millisecond
				period      = time.Second / 100
			)
			for _, st := range e.Calls {
				if !st.CPU {
					continue
				}
				cpuRequest := st.Duration
				for cpuRequest > 0 {
					if cpuRequest > jiffy {
						cpuDuration += jiffy
						cpuRequest -= jiffy
					} else {
						cpuDuration += cpuRequest
						cpuRequest = 0
					}

					if cpuDuration >= period {
						sample := st.Copy()
						sample.Duration = period
						p.Add(sample)
						cpuDuration -= period
					}
				}
			}
			return &p
		},
	},
}

type Profile struct {
	stacks []*StackTrace
}

func (p *Profile) String() string {
	var out string
	for _, st := range p.stacks {
		out += fmt.Sprintf("%s: %s\n", st.Duration, st.String())
	}
	return out
}

func (p *Profile) Add(sample *StackTrace) {
	for _, st := range p.stacks {
		if st.Equal(sample) {
			st.Duration += sample.Duration
			return
		}
	}
	p.stacks = append(p.stacks, sample.Copy())
}

func (p *Profile) Sort() {
	sort.Slice(p.stacks, func(i, j int) bool {
		return p.stacks[i].Duration > p.stacks[j].Duration
	})
}

// Random returns an random execution trace of the given duration and pool of
// stack traces.
func Random(seed int64, duration time.Duration, pool []*StackTrace) *ExecutionTrace {
	panic("TODO: implement")
}

// Repeat returns an execution trace of the given duration that consists of
// repetitions of the given stack traces.
func Repeat(duration time.Duration, list []*StackTrace) *ExecutionTrace {
	var e ExecutionTrace
	var offset time.Duration
	for {
		for _, st := range list {
			e.Calls = append(e.Calls, st)
			offset += st.Duration
			if offset >= duration {
				return &e
			}
		}
	}
}

type ExecutionTrace struct {
	Calls []*StackTrace
}

func (e *ExecutionTrace) Sample(offset time.Duration) *StackTrace {
	var ts time.Duration
	for _, st := range e.Calls {
		ts = ts + st.Duration
		if offset <= ts {
			return st
		}
	}
	return nil
}

func (e *ExecutionTrace) String() string {
	var out string
	var ts time.Duration
	for _, st := range e.Calls {
		out += fmt.Sprintf("%s: %s\n", ts, st.String())
		ts += st.Duration
	}
	out += fmt.Sprintf("%s: EXIT\n", ts)
	return out
}

type StackTrace struct {
	Duration time.Duration
	Frames   []string
	CPU      bool
}

func (s *StackTrace) String() string {
	prefix := "(off-cpu)"
	if s.CPU {
		prefix = "(on-cpu)"
	}
	return prefix + " " + strings.Join(s.Frames, ";")
}

func (s *StackTrace) Equal(o *StackTrace) bool {
	return reflect.DeepEqual(s.Frames, o.Frames) && s.CPU == o.CPU
}

func (s *StackTrace) Copy() *StackTrace {
	frames := make([]string, len(s.Frames))
	copy(frames, s.Frames)
	return &StackTrace{
		Duration: s.Duration,
		CPU:      s.CPU,
		Frames:   frames,
	}
}
