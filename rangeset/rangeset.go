// Package rangeset has set operations on ranges: Union, Intersect and Difference
package rangeset

import (
	"fmt"

	"github.com/biogo/store/step"
)

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type _bool bool

const (
	_true  = _bool(true)
	_false = _bool(false)
)

func (b _bool) Equal(e step.Equaler) bool {
	return b == e.(_bool)
}

// RangeSet acts like a bitvector where the interval values for true can be extracted.
type RangeSet struct {
	v *step.Vector
}

// Range is 0-based half-open.
type Range struct {
	Start, End int
}

func (i Range) String() string {
	return fmt.Sprintf("Range(%d-%d)", i.Start, i.End)
}

// Ranges returns the intervals that are set to true in the RangeSet.
func (b *RangeSet) Ranges() []Range {
	var posns []Range
	var start, end int
	var val step.Equaler
	var err error
	for end < b.v.Len() {
		start, end, val, err = b.v.StepAt(end)
		if err == nil {
			if !val.(_bool) {
				continue
			}
			posns = append(posns, Range{start, end})
		} else {
		}
	}
	return posns
}

// SetRange sets the values between start and end to true.
func (b *RangeSet) SetRange(start, end int) {
	b.v.SetRange(start, end, _true)
}

// ClearRange sets the values between start and end to false.
func (b *RangeSet) ClearRange(start, end int) {
	b.v.SetRange(start, end, _false)
}

// New returns a new RangeSet
func New(start, end int) (*RangeSet, error) {
	s, err := step.New(start, end, _false)
	if err == nil {
		s.Relaxed = true
	}
	return &RangeSet{v: s}, err
}

type operation int

const (
	intersectionOp operation = iota
	unionOp
	differenceOp
)

// Intersection returns a new RangeSet with intersecting regions from a and b.
func Intersection(a, b *RangeSet) *RangeSet {
	return combine(a, b, intersectionOp)
}

// Union returns a new RangeSet with the union of regions from a and b.
func Union(a, b *RangeSet) *RangeSet {
	return combine(a, b, unionOp)
}

// Difference returns a new RangeSet with regions in a that are absent from b.
func Difference(a, b *RangeSet) *RangeSet {
	return combine(a, b, differenceOp)
}

func combine(a, b *RangeSet, op operation) *RangeSet {
	if a.v.Zero != b.v.Zero {
		panic("intersection must have same Zero state for a and b")
	}
	c, err := New(0, max(a.v.Len(), b.v.Len()))
	if err != nil {
		panic(err)
	}
	var start, end int
	var val step.Equaler
	for end < a.v.Len() {
		start, end, val, _ = a.v.StepAt(end)
		if !val.(_bool) {
			continue
		}
		c.SetRange(start, end)
	}

	start, end = 0, 0

	for end < b.v.Len() {
		start, end, val, err = b.v.StepAt(end)
		if op == intersectionOp && !val.(_bool) {
			c.ClearRange(start, end)
			continue
		}
		if op == differenceOp && val.(_bool) {
			c.ClearRange(start, end)
			continue
		}
		if op == unionOp {
			if val.(_bool) {
				c.SetRange(start, end)
			}
			continue
		}
		c.v.ApplyRange(start, end, func(e step.Equaler) step.Equaler {
			if op == intersectionOp && (e.(_bool) && val.(_bool)) {
				return _bool(true)
			} else if op == differenceOp && (e.(_bool) && !val.(_bool)) {
				return _bool(true)
			}
			return _bool(false)
		})
	}
	return c
}
