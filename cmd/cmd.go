package main

import (
	"bufio"
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/anhk/exec"
	"k8s.io/client-go/util/homedir"
)

type MyWriteCloser struct {
	*bufio.Writer
}

func (mwc *MyWriteCloser) Close() error {
	// Noop
	return nil
}
func main() {
	cfgPath := "/root/.kube/config"
	if home := homedir.HomeDir(); home != "" {
		cfgPath = filepath.Join(home, ".kube", "config")
	}

	cli, err := exec.NewClient(cfgPath)
	if err != nil {
		panic(err)
	}

	if false {
		shell := cli.Shell("default", "bastion-log-0", "sh")
		if err := shell.Run(); err != nil {
			panic(err)
		}
	}

	if true {
		buf := bytes.NewBuffer([]byte{})
		file := cli.File("default", "bastion-relay-2", "/tmp/buddy.txt")
		err := file.ReadFileToWriter(&MyWriteCloser{bufio.NewWriter(buf)})
		if err != nil {
			panic(err)
		}
		fmt.Println(buf)
	}

	if false {
		data := "This is Good thing. \n"
		file := cli.File("default", "bastion-relay-2", "/tmp/buddy.txt")
		if err := file.WriteFileFromReader(strings.NewReader(data), int64(len(data))); err != nil {
			panic(err)
		}
	}
}
