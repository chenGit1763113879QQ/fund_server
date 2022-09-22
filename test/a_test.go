package test

import "testing"

type A struct {
	A1 string
	A2 string
	A3 []string
	A4 [][]string
	A5 string
	S1 []string
	S2 string
	S3 float64
	S6 float64
}

func Benchmark_struct(b *testing.B) {
	a := A{}
	for i := 0; i < b.N; i++ {
		p := a
		if p.A1 == "" {

		}
	}
}

func Benchmark_pointer(b *testing.B) {
	a := A{}
	for i := 0; i < b.N; i++ {
		p := &a
		if p.A1 == "" {

		}
	}
}
