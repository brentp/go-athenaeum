package windowmean_test

import (
	"fmt"
	"testing"

	"github.com/brentp/go-athenaeum/windowmean"
)

func TestMovingMean(t *testing.T) {
	a := []float64{1, 2, 3, 0, 0, 0, 2, 4, 2, 0, 0, 0, 0, 1, 1, 1, 0}
	b := windowmean.WindowMean(a, 1)
	exp := []float64{1.5, 2.0, 1.7, 1.0, 0.0, 0.7, 2.0, 2.7, 2.0, 0.7, 0.0, 0.0, 0.3, 0.7, 1.0, 0.7, 0.5}

	if fmt.Sprintf("%.1f", exp) != fmt.Sprintf("%.1f", b) {
		t.Fatalf("\nsent: %.1f\ngot : %.1f", exp, b)
	}
}

func TestMovingMeanUint16(t *testing.T) {
	a := []uint16{1, 2, 3, 0, 0, 0, 2, 4, 2, 0, 0, 0, 0, 1, 1, 1, 0}
	b := windowmean.WindowMeanUint16(a, 1)
	exp := []uint16{2, 2, 2, 1, 0, 1, 2, 3, 2, 1, 0, 0, 0, 1, 1, 1, 1}

	if fmt.Sprintf("%d", exp) != fmt.Sprintf("%d", b) {
		t.Fatalf("\nexpected: %d\ngot : %d", exp, b)
	}
}
