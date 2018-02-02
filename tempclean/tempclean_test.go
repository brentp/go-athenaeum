package tempclean_test

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/brentp/go-athenaeum/tempclean"
)

func TestCleanup(t *testing.T) {

	tmp, err := tempclean.TempFile("", "asdf")

	if err != nil {
		t.Fatal(err)
	}
	if tmp == nil {
		t.Fatal("error creating temp file")
	}

	tmp2, err := tempclean.TempFile("", "asdf")
	if err != nil {
		t.Fatal(err)
	}
	log.Println(tmp.Name(), "==", tmp2.Name())

	if filepath.Dir(tmp.Name()) != filepath.Dir(tmp2.Name()) {
		t.Fatal("expected directory re-use")
	}

}

func TestDifferentDirs(t *testing.T) {
	tmp, err := tempclean.TempFile("xx", "asdf")
	if err == nil {
		t.Fatal("expected error")
	}
	os.Mkdir("xx", 0777)
	tmp, err = tempclean.TempFile("xx", "asdf")
	if err != nil {
		t.Fatal("expected error")
	}

	tmp2, err := tempclean.TempFile("", "asdf")
	if err != nil {
		t.Fatal(err)
	}

	if filepath.Dir(tmp.Name()) == filepath.Dir(tmp2.Name()) {
		t.Fatal("expected different dirs")
	}

	log.Println(tmp.Name(), tmp2.Name())
	os.RemoveAll("xx")
}

func TestMain(m *testing.M) {
	m.Run()
	defer tempclean.Cleanup()
}
