package tests

import (
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func setup(files string) (path string, end func(), err error) {
	path, err = ioutil.TempDir("", "")

	if err == nil {
		end = func() {
			_ = os.RemoveAll(path)
		}
	} else {
		end = func() {}
		return
	}

	// Copy test files to temporary directory
	fileList, err := ioutil.ReadDir(files)
	if err != nil {
		return
	}

	var data []byte
	for _, file := range fileList {
		data, err = ioutil.ReadFile(filepath.Join(files, file.Name()))
		if err != nil {
			return
		}

		err = ioutil.WriteFile(filepath.Join(path, file.Name()), data, file.Mode())
		if err != nil {
			return
		}
	}

	return
}

func execute(path, client, args, stdin string) (output string, err error) {
	args = "-c " + filepath.Join(path, client) + ".yaml " + args
	ctx, done := context.WithTimeout(context.Background(), 10*time.Second)
	defer done()

	cmd := exec.CommandContext(ctx, "sporedb", strings.Split(args, " ")...)
	cmd.Dir = path
	cmd.Env = []string{"PASSWORD=1234"}
	cmd.Stdin = strings.NewReader(stdin)

	out, err := cmd.CombinedOutput()
	return string(out), err
}
