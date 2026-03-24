//go:build !solution

package fileleak

import (
	"errors"
	"fmt"
	"os"
)

type file struct {
	fd   string
	path string
}

type testingT interface {
	Errorf(msg string, args ...interface{})
	Cleanup(func())
}

func findFiles(t testingT) ([]file, error) {
	dirent, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		t.Errorf(err.Error())
		return []file(nil), err
	}

	files := make([]file, 0, len(dirent))

	for _, ent := range dirent {
		file_path, err := os.Readlink("/proc/self/fd/" + ent.Name())
		if file_path == "/proc/self/fd" {
			continue
		}

		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Errorf(err.Error())
			return []file(nil), err
		}
		files = append(files, file{ent.Name(), file_path})
	}
	return files, nil
}

func VerifyNone(t testingT) {

	files, err := findFiles(t)
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	startFiles := make(map[string]string, len(files))
	for _, f := range files {
		startFiles[f.fd] = f.path
	}

	t.Cleanup(func() {
		endFiles, err := findFiles(t)
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		for _, f := range endFiles {
			p, ok := startFiles[f.fd]
			if !ok || p != f.path {
				t.Errorf(fmt.Sprintf("err"))
				return
			}
		}
	})
}
