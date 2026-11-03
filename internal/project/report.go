package project

import "fmt"

// Report collects the outcome of an operation. Problems make the command fail;
// warnings are informational.
type Report struct {
	Actions  []string
	Warnings []string
	Problems []string
	Hints    []string
}

func (r *Report) Hint(format string, args ...any) {
	r.Hints = append(r.Hints, fmt.Sprintf(format, args...))
}

func (r *Report) Action(format string, args ...any) {
	r.Actions = append(r.Actions, fmt.Sprintf(format, args...))
}

func (r *Report) Warn(format string, args ...any) {
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, args...))
}

func (r *Report) Problem(format string, args ...any) {
	r.Problems = append(r.Problems, fmt.Sprintf(format, args...))
}

func (r *Report) Merge(o *Report) {
	if o == nil {
		return
	}
	r.Actions = append(r.Actions, o.Actions...)
	r.Warnings = append(r.Warnings, o.Warnings...)
	r.Problems = append(r.Problems, o.Problems...)
	r.Hints = append(r.Hints, o.Hints...)
}

func (r *Report) Failed() bool { return len(r.Problems) > 0 }
