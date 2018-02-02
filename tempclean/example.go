// +build ignore

package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/brentp/go-athenaeum/tempclean"
)

func main() {

	// required in main())
	defer tempclean.Cleanup()
	var tmp *os.File
	var err error

	// create tempfiles in a subdirecotry of the default TMPDIR
	tmp, err = tempclean.TempFile("lumpy-smoother", ".vcf.gz")

	tmp, err = tempclean.TempFile("prefix", "suffix")
	if err != nil {
		log.Fatal(err)
	}

	tmp2, err := tempclean.TempFile("aprefix", "asuffix")
	if err != nil {
		log.Fatal(err)
	}

	if filepath.Dir(tmp.Name()) != filepath.Dir(tmp2.Name()) {
		log.Fatal("expected same path", tmp.Name(), " ", tmp2.Name())
	}

	// can also create a tmp sub-directory and make files inside of it:
	tmpD, err := tempclean.TempDir("./", "asdf")
	if err != nil {
		log.Fatal(err)
	}

	tmpF, err := tmpD.TempFile("my-file-prefix", ".txt")
	if err != nil {
		log.Fatal(err)
	}

	if !strings.HasSuffix(tmpF.Name(), ".txt") {
		panic("expected .txt suffix, got:" + tmpF.Name())
	}

	// note that we can't recover a log.Fatal, but can still get a panic
	// YES
	panic("i can still cleanup")
	// NO
	// log.Fatal("i can't cleanup")
	// OK
	// tempclean.Cleanup(); log.Fatal("manual")
}
