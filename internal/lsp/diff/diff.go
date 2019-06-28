// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package diff implements the Myers diff algorithm.
package diff

import "strings"
//import "fmt"
import "os"

// Sources:
// https://blog.jcoglan.com/2017/02/17/the-myers-diff-algorithm-part-3/
// https://www.codeproject.com/Articles/42279/%2FArticles%2F42279%2FInvestigating-Myers-diff-algorithm-Part-1-of-2

type Op struct {
	Kind    OpKind
	Content []string // content from b
	I1, I2  int      // indices of the line in a
	J1      int      // indices of the line in b, J2 implied by len(Content)
}

type OpKind int

const (
	Delete OpKind = iota
	Insert
	Equal
)

func (k OpKind) String() string {
	switch k {
	case Delete:
		return "delete"
	case Insert:
		return "insert"
	case Equal:
		return "equal"
	default:
		panic("unknown operation kind")
	}
}

func ApplyEdits(a []string, operations []*Op) []string {
	var b []string
	var prevI2 int
	for _, op := range operations {
		// catch up to latest indices
		if op.I1-prevI2 > 0 {
			for _, c := range a[prevI2:op.I1] {
				b = append(b, c)
			}
		}
		switch op.Kind {
		case Equal, Insert:
			b = append(b, op.Content...)
		}
		prevI2 = op.I2
	}
	// final catch up
	if len(a)-prevI2 > 0 {
		for _, c := range a[prevI2:len(a)] {
			b = append(b, c)
		}
	}
	return b
}

func log2File(text string) {
	f, err := os.OpenFile("C:\\stupid.txt", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}

	defer f.Close()

	if _, err = f.WriteString(text); err != nil {
		panic(err)
	}
}

// stringEqualIgnoreLF compare strings ignore the line feet different, \r\n, \n
func stringEqualIgnoreLF(a,b string) bool {
	la := len(a) - 1
	lb := len(b) - 1
	if la > 0 && a[la-1] == '\r' {
		la = la -1
	}

	if lb > 0 && b[lb-1] == '\r' {
		lb = lb -1
	}

	if la != lb {
		return false
	}

	for i := 0; i < la; i++ {
		if (a[i] != b[i]) {
			return false
		}
	}

	return true
}

func myOperations(a, b []string) []*Op {
	M := len(a)
	var i int
	solution := make([]*Op, len(a)+len(b))

	aIdx := 0
	for bIdx, bContent := range b {
		// if not the same, find out the same line of a
		if ( aIdx < M && !stringEqualIgnoreLF(bContent,a[aIdx])) {
			//log2File(fmt.Sprintf("bContent:%s\n", bContent))
			//log2File(fmt.Sprintf("a[aIdx]:%s\n", a[aIdx]))
			prv := aIdx

			// find the same line from a
			for ; aIdx < M; aIdx++ {
				if stringEqualIgnoreLF(bContent,a[aIdx]) {
					break
				}
			}

			// delete [prv:aIdx] from a
			op1 := &Op{}
			op1.Kind = Delete
			op1.I1 = prv
			op1.I2 = aIdx

			solution[i] = op1
			i++

			//log2File(fmt.Sprintf("Delete, I1:%d, I2:%d\n", prv, aIdx))
		}

		if (aIdx >= M) {
			// insert all remain lines of b into a
			op2 := &Op{}
			op2.Kind = Insert
			op2.I1 = bIdx
			op2.I2 = bIdx
			op2.J1 = bIdx
			op2.Content = b[bIdx:]

			solution[i] = op2
			i++
			//log2File(fmt.Sprintf("Insert, I1:%d, I2:%d, J1:%d,\n", bIdx, bIdx, bIdx))
			break
		}

		aIdx++
	}
	return solution[:i]
}

// Operations returns the list of operations to convert a into b, consolidating
// operations for multiple lines and not including equal lines.
func Operations(a, b []string) []*Op {
	trace, offset := shortestEditSequence(a, b)
	snakes := backtrack(trace, len(a), len(b), offset)

	M, N := len(a), len(b)

	var i int
	solution := make([]*Op, len(a)+len(b))

	add := func(op *Op, i2, j2 int) {
		if op == nil {
			return
		}
		op.I2 = i2
		if op.Kind == Insert {
			op.Content = b[op.J1:j2]
			//log2File(fmt.Sprintf("Insert, I1:%d, I2:%d, J1:%d, lines:%d\n", op.I1, op.I2, op.J1, len(op.Content)))
		} else {
			//log2File(fmt.Sprintf("Delete, I1:%d, I2:%d\n", op.I1, op.I2))
		}

		solution[i] = op
		i++
	}
	x, y := 0, 0
	for _, snake := range snakes {
		if len(snake) < 2 {
			continue
		}
		var op *Op
		// delete (horizontal)
		for snake[0]-snake[1] > x-y {
			if op == nil {
				op = &Op{
					Kind: Delete,
					I1:   x,
					J1:   y,
				}
			}
			x++
			if x == M {
				break
			}
		}
		add(op, x, y)
		op = nil
		// insert (vertical)
		for snake[0]-snake[1] < x-y {
			if op == nil {
				op = &Op{
					Kind: Insert,
					I1:   x,
					J1:   y,
				}
			}
			y++
		}
		add(op, x, y)
		op = nil
		// equal (diagonal)
		for x < snake[0] {
			x++
			y++
		}
		if x >= M && y >= N {
			break
		}
	}
	return solution[:i]
}

// backtrack uses the trace for the edit sequence computation and returns the
// "snakes" that make up the solution. A "snake" is a single deletion or
// insertion followed by zero or diagnonals.
func backtrack(trace [][]int, x, y, offset int) [][]int {
	snakes := make([][]int, len(trace))
	d := len(trace) - 1
	for ; x > 0 && y > 0 && d > 0; d-- {
		V := trace[d]
		if len(V) == 0 {
			continue
		}
		snakes[d] = []int{x, y}

		k := x - y

		var kPrev int
		if k == -d || (k != d && V[k-1+offset] < V[k+1+offset]) {
			kPrev = k + 1
		} else {
			kPrev = k - 1
		}

		x = V[kPrev+offset]
		y = x - kPrev
	}
	if x < 0 || y < 0 {
		return snakes
	}
	snakes[d] = []int{x, y}
	return snakes
}

// shortestEditSequence returns the shortest edit sequence that converts a into b.
func shortestEditSequence(a, b []string) ([][]int, int) {
	M, N := len(a), len(b)
	V := make([]int, 2*(N+M)+1)
	offset := N + M
	trace := make([][]int, N+M+1)

	// Iterate through the maximum possible length of the SES (N+M).
	for d := 0; d <= N+M; d++ {
		copyV := make([]int, len(V))
		// k lines are represented by the equation y = x - k. We move in
		// increments of 2 because end points for even d are on even k lines.
		for k := -d; k <= d; k += 2 {
			// At each point, we either go down or to the right. We go down if
			// k == -d, and we go to the right if k == d. We also prioritize
			// the maximum x value, because we prefer deletions to insertions.
			var x int
			if k == -d || (k != d && V[k-1+offset] < V[k+1+offset]) {
				x = V[k+1+offset] // down
			} else {
				x = V[k-1+offset] + 1 // right
			}

			y := x - k

			// Diagonal moves while we have equal contents.
			for x < M && y < N && stringEqualIgnoreLF(a[x], b[y]) {
				x++
				y++
			}

			V[k+offset] = x

			// Return if we've exceeded the maximum values.
			if x == M && y == N {
				// Makes sure to save the state of the array before returning.
				copy(copyV, V)
				trace[d] = copyV
				return trace, offset
			}
		}

		// Save the state of the array.
		copy(copyV, V)
		trace[d] = copyV
	}
	return nil, 0
}

func SplitLines(text string) []string {
	lines := strings.SplitAfter(text, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
