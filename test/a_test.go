package test

import (
	"fmt"
	"testing"

	"github.com/bytedance/sonic"
	"gonum.org/v1/gonum/mat"
)

var a struct {
	SleepSet int
	openWrt  string
}

func Benchmark_struct(b *testing.B) {
	fmt.Println(sonic.MarshalString(a))

	arr1 := make([]float64, 100)
	arr2 := make([]float64, 100)
	for i := range arr1 {
		arr1[i] = float64(i)
		arr2[i] = float64(i * i)
	}

	for i := 0; i < b.N; i++ {
		for i := range arr1 {
			arr1[i] *= 0.5
			arr2[i] *= 0.5
		}
	}
}

func Benchmark_pointer(b *testing.B) {
	arr1 := make([]float64, 100)
	arr2 := make([]float64, 100)
	for i := range arr1 {
		arr1[i] = float64(i)
		arr2[i] = float64(i * i)
	}
	matrix := mat.NewDense(2, 100, append(arr1, arr2...))

	for i := 0; i < b.N; i++ {
		matrix.Scale(0.5, matrix)
	}
}
