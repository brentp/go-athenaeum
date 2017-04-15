package unsplit

import (
	"bytes"
	"reflect"
	"testing"
)

func TestUnSplit(t *testing.T) {
	u := New(sentence, []byte{'\t'})
	exp := bytes.Split(sentence, []byte{'\t'})
	var obs [][]byte
	for {

		tok := u.Next()
		if tok == nil {
			break
		}
		obs = append(obs, tok)
	}

	if !reflect.DeepEqual(exp, obs) {
		t.Fatalf("expected: %s, got: %s", exp, obs)
	}
}

var sentence = []byte("oh hello\tmyxx\tname\tis\tvery\texciting.\tdo\tyou\tagree?\tgounsplit\tgounsplitgounsplitgounsplit\tbye.")

func BenchmarkUnsplit(b *testing.B) {

	for i := 0; i < b.N; i++ {
		u := New(sentence, []byte{'\t'})
		for {
			tok := u.Next()
			if tok == nil {
				break
			}
		}
	}
}

func BenchmarkSplit(b *testing.B) {

	for i := 0; i < b.N; i++ {
		toks := bytes.Split(sentence, []byte{'\t'})
		_ = toks[1]
	}
}

func BenchmarkSplitN(b *testing.B) {
	n := bytes.Count(sentence, []byte{'\t'})

	for i := 0; i < b.N; i++ {
		toks := bytes.SplitN(sentence, []byte{'\t'}, n)
		_ = toks[1]
	}
}
