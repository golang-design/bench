// Copyright 2020 The golang.design Initiative Authors.
// All rights reserved. Use of this source code is governed
// by a GNU GPLv3 license that can be found in the LICENSE file.

package example

import "testing"

func grow(n int) {
	if n > 0 {
		grow(n - 1)
	}
}

func BenchmarkDemo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		grow(10000) // try change this and re-run `bench`
	}
}
