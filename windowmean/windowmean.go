// Package windowmean provides a moderately efficient implementation of a movingmean.
package windowmean

// WindowMean gives the window-based mean of a slice of float64's
// using a window of the given radius.
func WindowMean(a []float64, radius int) []float64 {
	var s float64
	b := make([]float64, len(a))
	for i := 0; i < radius; i++ {
		s += a[i]
	}
	w := float64(2*(radius+1) - 1)
	iw := float64(radius)
	for i := radius; i < int(w); i++ {
		s += a[i]
		iw++
		b[i-radius] = s / iw
	}

	for i := radius + 1; i < len(a)-radius; i++ {
		b[i-radius] = s / w
		s -= a[i-radius-1]
		s += a[i+radius]
	}
	for i := len(a) - int(radius); i <= len(a); i++ {
		b[i-1] = s / w
		s -= a[i-radius-1]
		w--
	}
	return b
}

// WindowMeanUint16 gives the window-based mean of a slice of uint16's
// using a window of the given radius.
func WindowMeanUint16(a []uint16, radius int) []uint16 {
	var s float64
	b := make([]uint16, len(a))
	for i := 0; i < radius; i++ {
		s += float64(a[i])
	}
	w := float64(2*(radius+1) - 1)
	iw := float64(radius)
	for i := radius; i < int(w); i++ {
		s += float64(a[i])
		iw++
		b[i-radius] = uint16(0.5 + s/float64(iw))
	}

	for i := radius + 1; i < len(a)-radius; i++ {
		b[i-radius] = uint16(0.5 + s/w)
		s -= float64(a[i-radius-1])
		s += float64(a[i+radius])
	}
	for i := len(a) - int(radius); i <= len(a); i++ {
		b[i-1] = uint16(0.5 + s/w)
		s -= float64(a[i-radius-1])
		w--
	}
	return b
}
