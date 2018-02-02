// tempclean implements a TempFile that makes a best-effort to clean up at exit
package tempclean

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"
)

type directories struct {
	*sync.Mutex
	dirs map[string]string
}

var d directories

var tmpdir *TmpDir

func cleanup() {
	d.Lock()

	for _, dir := range d.dirs {
		if err := os.RemoveAll(dir); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		delete(d.dirs, dir)
	}
	d.Unlock()
}

func init() {
	d = directories{Mutex: &sync.Mutex{}, dirs: make(map[string]string, 5)}

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM, os.Interrupt, syscall.SIGQUIT, syscall.SIGABRT)
		<-c
		cleanup()
		os.Exit(1)
	}()
	var err error
	tmpdir, err = TempDir("", DirPrefix)
	if err != nil {
		panic(err)
	}

}

// Cleanup must be called via defer inside of main.
// func main() {
//    defer tempclean.Cleanup()
// }
func Cleanup() {
	if err := recover(); err != nil {
		cleanup()
		panic(err)
	}
	cleanup()
}

// key is the directory argument given to the TempFile/TempDir
// functions. It creates a new sub-dir for each new directory
// and registers it for deletion.
func (d *directories) get(dir, prefix string) (name string, err error) {
	key := dir + "::" + prefix
	d.Lock()
	defer d.Unlock()
	if tmpdir, ok := d.dirs[key]; ok {
		return tmpdir, nil
	}

	name, err = ioutil.TempDir(dir, prefix)
	/*
		if err != nil {
			if _, ok := err.(*os.PathError); ok {
				err = os.MkdirAll(key, 0777)
				name = key
			}
		}
	*/
	if err == nil {
		d.dirs[key] = name
	}
	return name, err
}

var DirPrefix = "tempclean-"

type TmpDir struct {
	path  string
	files []string
}

// TempDir creates a new temp directory using ioutil.TempDir and registers it for cleanup when the program exits.
func TempDir(dir, prefix string) (t *TmpDir, err error) {
	base, err := d.get(dir, prefix)
	if err != nil {
		return nil, err
	}
	return &TmpDir{path: base, files: make([]string, 0, 20)}, nil
}

func rm(f *os.File) {
	_ = os.Remove(f.Name())
}

func (t *TmpDir) Remove() error {
	return os.RemoveAll(t.path)
}

// TempFile creates a new temp file using ioutil.TempFile and registers it for cleanup when the program exits
// if `dir` does not exist, it will be created and used as the base directory in which temp files are created.
func TempFile(prefix, suffix string) (f *os.File, err error) {
	return tmpdir.TempFile(prefix, suffix)
}

// TempFile creates a new temp file in the directory.
func (t *TmpDir) TempFile(prefix, suffix string) (f *os.File, err error) {
	log.Println("prefix:", prefix, " path:", t.path)
	f, err = iTempFile(t.path, prefix, suffix)
	if err == nil {
		runtime.SetFinalizer(f, rm)
		t.files = append(t.files, f.Name())
	}
	return f, err
}

// modified from golang's ioutil

var rand uint32
var randmu sync.Mutex

func reseed() uint32 {
	return uint32(time.Now().UnixNano() + int64(os.Getpid()))
}

func nextSuffix() string {
	randmu.Lock()
	r := rand
	if r == 0 {
		r = reseed()
	}
	r = r*1664525 + 1013904223 // constants from Numerical Recipes
	rand = r
	randmu.Unlock()
	return strconv.Itoa(int(1e9 + r%1e9))[1:]
}

func iTempFile(dir, prefix, suffix string) (f *os.File, err error) {
	if dir == "" {
		dir = os.TempDir()
	}

	nconflict := 0
	for i := 0; i < 10000; i++ {
		name := filepath.Join(dir, prefix+nextSuffix()+suffix)
		f, err = os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
		if os.IsExist(err) {
			if nconflict++; nconflict > 10 {
				randmu.Lock()
				rand = reseed()
				randmu.Unlock()
			}
			continue
		}
		break
	}
	return
}
