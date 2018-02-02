// tempclean implements a TempFile that makes a best-effort to clean up at exit
package tempclean

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
)

type directories struct {
	*sync.Mutex
	dirs map[string]string
}

var d directories

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
func (d *directories) get(key string) (name string, err error) {
	d.Lock()
	defer d.Unlock()
	if tmpdir, ok := d.dirs[key]; ok {
		return tmpdir, nil
	}

	name, err = ioutil.TempDir(key, "")
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

// TempDir creates a new temp directory using ioutil.TempDir and registers it for cleanup when the program exits.
func TempDir(dir, prefix string) (name string, err error) {
	base, err := d.get(dir)
	if err != nil {
		return base, err
	}
	return ioutil.TempDir(base, prefix)
}

func rm(f *os.File) {
	os.Remove(f.Name())
}

// TempFile creates a new temp file using ioutil.TempFile and registers it for cleanup when the program exits
// if `dir` does not exist, it will be created and used as the base directory in which temp files are created.
func TempFile(dir, prefix string) (f *os.File, err error) {
	base, derr := d.get(dir)
	if derr != nil {
		return nil, derr
	}
	f, err = ioutil.TempFile(base, prefix)
	if err == nil {
		runtime.SetFinalizer(f, rm)
	}
	return f, err
}
