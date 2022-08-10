package main

import (
	"path/filepath"

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
	shell := cli.Shell("default", "bastion-log-0", "sh")
	if err := shell.Run(); err != nil {
		panic(err)
	}
}
