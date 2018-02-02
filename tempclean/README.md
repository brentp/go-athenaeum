Like ioutil.TempFile/TempDir, but clean up files at exit.

If you don't use `log.Fatal`, this should cleanup tempfiles as they go out of scope
and clean up directories at exit, even when there is an error.

```Go
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

	tmpD, err := tempclean.TempFile("relative", "asdf")
	if err != nil {
		log.Fatal(err)
	}

	if filepath.Dir(tmpD.Name()) != "relative" {
		log.Fatal(filepath.Dir(tmpD.Name()))
	}
	// note that we can't recover a log.Fatal, but can still get a panic
	// YES
	panic("i can still cleanup")
	// NO
	// log.Fatal("i can't cleanup")
	// OK
	// tempclean.Cleanup(); log.Fatal("manual")
}
```
