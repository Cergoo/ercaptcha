// Copyright 2013 ercaptcha  All rights reserved.
// Use of this source code is governed by a BSD-style

package main

import (
	"math/rand"
	"sync/atomic"
)

/*
counter limit
create example:	&counter{limit: limit}
*/
type (
	counter struct {
		counter uint32
		limit   uint32
	}
)

func (t *counter) inc() uint32 {
	if count := atomic.AddUint32(&t.counter, 1); count > t.limit {
		atomic.StoreUint32(&t.counter, 0)
	}
	return t.counter
}

/*
base + delta random
*/
func rnd_int(base, dr int) int {
	return base + rand.Intn(dr)
}
