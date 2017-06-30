package glosh

import (
	"log"
	"math/rand"
	"sort"
)

// LSH ecompasses the methods for locality sensitive hashing
type LSH struct {
	embedding []embedding
	hash      map[uint64][]int
	vectors   [][]float64
	options   Options
	// sampling of distances
	distances []float64
}

// Options for LSH
type Options struct {
	// Distance function for the given vectors
	Dist func(a, b []float64) float64
	// Number of times to sample distance in the input vectors
	Samples int
}

// New returns an LSH for the given vectors with n embeddings each using d bits.
func New(vectors [][]float64, n int, d int, opts Options) LSH {
	l := LSH{embedding: make([]embedding, n), vectors: vectors, options: opts}
	for i := 0; i < n; i++ {
		l.embedding[i] = l.newEmbedding(d)
	}

	l.hash = make(map[uint64][]int)
	for id, vector := range vectors {
		for _, embedding := range l.embedding {
			h := embedding.embed(vector)
			l.hash[h] = append(l.hash[h], id)
			// for each bit in the hashed value, we flip it and use
			// the mutated hash-value as a key to (also) store the index.
			// this is a poor-man's multi-probe.
			for m := 0; m < len(embedding); m += 2 {
				mut := h ^ (1 << uint64(m))
				l.hash[mut] = append(l.hash[mut], id)
			}
		}
	}
	for i, ids := range l.hash {
		l.hash[i] = deduplicate(ids)
	}
	l.MapStats()
	l.sample()
	return l
}

func deduplicate(ids []int) []int {
	hash := make(map[int]struct{}, len(ids)/2)
	result := make([]int, 0, len(ids))
	var empty struct{}
	for _, id := range ids {
		if _, ok := hash[id]; !ok {
			result = append(result, id)
			hash[id] = empty
		}
	}
	return result
}

func (l *LSH) sample() {
	l.distances = make([]float64, l.options.Samples)
	for i := range l.distances {
		ai := rand.Intn(len(l.vectors))
		bi := rand.Intn(len(l.vectors))
		l.distances[i] = l.options.Dist(l.vectors[ai], l.vectors[bi])
	}
	sort.Float64s(l.distances)
}

func (l *LSH) MapStats() {
	ones := 0
	for _, v := range l.hash {
		if len(v) == 1 {
			ones++
		}
	}
	log.Printf("%d/%d (%.1f%%) have a single value.", ones, len(l.hash), 100*float64(ones)/float64(len(l.hash)))
}

// Distance returns the distance at percentile p where p
// ranges from 0 to 100
func (l LSH) Distance(p float64) float64 {
	if len(l.distances) == 0 {
		panic("glosh: must set opts.Samples to allow using distance")
	}
	idx := int(0.5 + p/100*float64(len(l.distances)))
	return l.distances[idx]
}

// ANN returns the k (approximate) nearest neighbors to the vector q
// constrained to be within maxDist.
func (l LSH) ANN(q []float64, k int, maxDist float64) []Neighbor {
	hits := make([]Neighbor, 0, k)
	for id := range l.candidates(q) {
		d := l.options.Dist(q, l.vectors[id])
		if d > maxDist {
			continue
		}
		hits = append(hits, Neighbor{ID: id, Dist: d})
		if len(hits) == k {
			break
		}
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].Dist < hits[j].Dist })
	return hits
}

// return a list of candidates with the same has value under any of the embeddings.
func (l LSH) candidates(q []float64) (candidates map[int]bool) {
	candidates = make(map[int]bool, 100)
	for _, emb := range l.embedding {
		h := emb.embed(q)
		for _, c := range l.hash[h] {
			if _, ok := candidates[c]; !ok {
				candidates[c] = true
			}
		}
	}
	/*
		if len(candidates) < 5 || len(candidates) > 5000 {
			log.Println(len(candidates))
		} else {
			log.Println("OK")
		}
	*/
	return candidates
}

type embedding [][]float64

func (l *LSH) newEmbedding(d int) embedding {
	cols := len(l.vectors[0])
	normals := make([][]float64, d)
	for i := 0; i < d; i++ {
		normals[i] = normal(cols)
	}
	return embedding(normals)
}

func normal(cols int) []float64 {
	result := make([]float64, cols)
	for col := 0; col < cols; col++ {
		result[col] = rand.NormFloat64()
	}
	return result
}

func (e embedding) embed(vector []float64) uint64 {
	var result uint64
	for i, em := range e {
		if dot(vector, em) > 0 {
			result |= (1 << uint64(i))
		}
	}

	return result
}

func dot(x, y []float64) float64 {
	var sum float64
	for i, v := range x {
		sum += y[i] * v
	}
	return sum
}

// Neighbor is returned from `ANN`. It holds the id and distance to the query.
type Neighbor struct {
	// Id indicates the index into the vectors sent to LSH
	ID int
	// Dist is the distance to the query
	Dist float64
}
