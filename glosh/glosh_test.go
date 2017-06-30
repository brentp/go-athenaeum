package glosh_test

import (
	"log"
	"math"
	"math/rand"
	"testing"

	"github.com/brentp/glosh"
)

func dist(v1, v2 []float64) float64 {
	var s float64
	for i, v := range v1 {
		s += math.Abs(v - v2[i])
	}
	return s
}

func randomVector(n int) []float64 {
	v := make([]float64, n)
	for i := range v {
		v[i] = rand.NormFloat64()
	}
	return v
}

func BenchmarkLSH(b *testing.B) {
	vectors := make([][]float64, 1000000)
	cols := 200
	count := 30
	for i := range vectors {
		vectors[i] = randomVector(cols)
	}

	r := glosh.New(vectors, 5, 15, glosh.Options{Dist: dist, Samples: 100000})
	for i := 1; i < 10; i++ {
		log.Println(i, r.Distance(float64(i)))
	}

	maxDist := r.Distance(5)
	b.ResetTimer()
	/*
	   sames := 0
	   tests := 0
	*/
	missed := 0
	for i := 0; i < b.N; i++ {
		for k, v := range vectors {
			if k%5000 == 0 {
				log.Println(i, k, missed)
			}
			hits := r.ANN(v, count, maxDist)
			if len(hits) < 30 {
				missed++
			}
		}
	}
}
