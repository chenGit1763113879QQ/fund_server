package test

import (
	"testing"
)

func Benchmark1(b *testing.B) {
	for i := 0; i < b.N; i++ {
	}
}

func Benchmark2(b *testing.B) {
	for i := 0; i < b.N; i++ {
	}
}
