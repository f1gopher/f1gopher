// F1Gopher - Copyright (C) 2023 f1gopher
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package panel

import (
	"sync"
)

type circularBuffer[T any] struct {
	values []T
	start  int
	end    int
	size   int
	lock   sync.RWMutex
}

func createCircularBuffer[T any](size int) *circularBuffer[T] {
	return &circularBuffer[T]{
		values: make([]T, size),
		start:  0,
		end:    0,
		size:   size,
	}
}

func (c *circularBuffer[T]) reset() {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.start = 0
	c.end = -1
}

func (c *circularBuffer[T]) count() int {
	if c.end == -1 {
		return 0
	}

	if c.end < c.start {
		return (c.size - c.start) + c.end
	}

	return (c.end - c.start) + 1
}

func (c *circularBuffer[T]) add(value T) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.end == -1 {
		c.end = 0
		c.values[c.end] = value
		return
	}

	c.end++
	if c.end >= c.size {
		c.end = 0
	}

	if c.end == c.start {
		c.start++
		if c.start >= c.size {
			c.start = 0
		}
	}

	c.values[c.end] = value
}

func (c *circularBuffer[T]) get(index int) T {
	c.lock.RLock()
	defer c.lock.RUnlock()

	actualIndex := c.start + index
	if actualIndex >= c.size {
		actualIndex = index - (c.size - c.start)
	}

	return c.values[actualIndex]
}
