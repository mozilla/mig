package yara

import (
	"strconv"
	"sync"
)

/*
The closure type stores (pointers to) arbitrary data, returning a
(usually small) uintptr. The uintptr value can be passed through C
code to exported callback functions written in Go that can use it to
access the data without violating the rules for passing pointers
through C code.

Concurrent access to the stored data is protected through a
sync.RWMutex.
*/
type closure struct {
	m map[uintptr]interface{}
	sync.RWMutex
}

func (c *closure) Put(elem interface{}) uintptr {
	c.Lock()
	if c.m == nil {
		c.m = make(map[uintptr]interface{})
	}
	defer c.Unlock()
	for i := uintptr(0); ; i++ {
		_, ok := c.m[i]
		if !ok {
			c.m[i] = elem
			return i
		}
	}
}

func (c *closure) Get(id uintptr) interface{} {
	c.RLock()
	defer c.RUnlock()
	if r, ok := c.m[id]; ok {
		return r
	}
	panic("get: element " + strconv.Itoa(int(id)) + " not found")
}

func (c *closure) Delete(id uintptr) {
	c.Lock()
	defer c.Unlock()
	if _, ok := c.m[id]; !ok {
		panic("delete: element " + strconv.Itoa(int(id)) + " not found")
	}
	delete(c.m, id)
}

var callbackData closure
