// Copyright 2020 CUE Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package runtime

import (
	"sync"
)

// index maps conversions from label names to internal codes.
//
// All instances belonging to the same package should share this index.
type index struct {
	labelMap map[string]int
	labels   []string

	offset int
	parent *index

	mutex     sync.Mutex
	typeCache sync.Map // map[reflect.Type]evaluated
}

// work around golang-ci linter bug: fields are used.
func init() {
	var i index
	i.mutex.Lock()
	i.mutex.Unlock()
	i.typeCache.Load(1)
}

// sharedIndex is used for indexing builtins and any other labels common to
// all instances.
var sharedIndex = newSharedIndex()

func newSharedIndex() *index {
	i := &index{
		labelMap: map[string]int{"": 0},
		labels:   []string{"_"},
	}
	return i
}

// newIndex creates a new index.
func newIndex(parent *index) *index {
	i := &index{
		labelMap: map[string]int{},
		offset:   len(parent.labels) + parent.offset,
		parent:   parent,
	}
	return i
}

func (x *index) IndexToString(i int64) string {
	for ; int(i) < x.offset; x = x.parent {
	}
	return x.labels[int(i)-x.offset]
}

func (x *index) StringToIndex(s string) int64 {
	for p := x; p != nil; p = p.parent {
		if f, ok := p.labelMap[s]; ok {
			return int64(f)
		}
	}
	index := len(x.labelMap) + x.offset
	x.labelMap[s] = index
	x.labels = append(x.labels, s)
	return int64(index)
}
