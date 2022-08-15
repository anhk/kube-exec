package main

import (
	"path/filepath"
	"strings"

	"github.com/anhk/exec"
	"k8s.io/client-go/util/homedir"
)

func main() {
	cfgPath := "/root/.kube/config"
	if home := homedir.HomeDir(); home != "" {
		cfgPath = filepath.Join(home, ".kube", "config")
	}

	cli, err := exec.NewClient(cfgPath)
	if err != nil {
		panic(err)
	}
	//shell := cli.Shell("default", "bastion-log-0", "sh")
	//if err := shell.Run(); err != nil {
	//	panic(err)
	//}

	//file := cli.File("default", "bastion-relay-2", "/tmp/buddy.txt")
	//reader, err := file.ReadFile()
	//if err != nil {
	//	panic(err)
	//}
	//defer reader.Close()
	//
	//data, err := ioutil.ReadAll(reader)
	//if err != nil {
	//	panic(err)
	//}
	//fmt.Println(string(data))

	file := cli.File("default", "bastion-relay-2", "/tmp/buddy.txt")
	if err := file.WriteFile(strings.NewReader("Hello world tt3a4t\n")); err != nil {
		panic(err)
	}
}
