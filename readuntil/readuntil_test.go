package readuntil

import (
	"io"
	"io/ioutil"
	"strings"
	"testing"
)

func TestRead(t *testing.T) {
	str := "hello how are you cruel world"
	s := strings.NewReader(str)

	n, err := Index(s, []byte("cruel"))
	if err != nil {
		t.Fatal("expected nil error for found pattern")
	}
	b, err := ioutil.ReadAll(s)
	if err != nil {
		t.Fatal(err)
	}

	if string(b) != "cruel world" {
		t.Fatal("didn't seek to expected spot")
	}
	if str[n:] != "cruel world" {
		t.Fatal("didn't return expected value")
	}

}

func TestEnd(t *testing.T) {
	str := "hello how are you cruel world!"
	s := strings.NewReader(str)

	n, err := Index(s, []byte("!"))
	if err != nil {
		t.Fatal("expected nil error for found pattern")
	}
	if n != len(str)-1 {
		t.Fatal("expected to get to end of string")
	}
}

func TestNotFound(t *testing.T) {
	str := "hello how are you cruel world"
	s := strings.NewReader(str)

	_, err := Index(s, []byte("xxcruel"))
	if err != io.EOF {
		t.Fatalf("expected EOF for non existent pattern, got: %s", err)
	}
}
