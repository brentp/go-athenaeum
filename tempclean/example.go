// +build ignore

package main

import (
	"log"
	"path/filepath"

	"github.com/brentp/go-athenaeum/tempclean"
)

func main() {

	// required in main())
	defer tempclean.Cleanup()

	tmp, err := tempclean.TempFile("", "asdf")
	if err != nil {
		log.Fatal(err)
	}

	tmp2, err := tempclean.TempFile("", "adsf")
	if err != nil {
		log.Fatal(err)
	}

	if filepath.Dir(tmp.Name()) != filepath.Dir(tmp2.Name()) {
		log.Fatal("expected same path")
	}

	tmpD, err := tempclean.TempFile("./", "asdf")
	if err != nil {
		log.Fatal(err)
	}

	if filepath.Dir(filepath.Dir(tmpD.Name())) != "." {
		panic(filepath.Dir(filepath.Dir(tmpD.Name())))
	}
	// note that we can't recover a log.Fatal, but can still get a panic
	// YES
	panic("i can still cleanup")
	// NO
	// log.Fatal("i can't cleanup")
	// OK
	// tempclean.Cleanup(); log.Fatal("manual")
}
