/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cp

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/anhk/exec/exec"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

// CopyOptions have the data required to perform the copy operation
type CopyOptions struct {
	Container  string
	Namespace  string
	NoPreserve bool
	MaxTries   int

	ClientConfig *restclient.Config
	Clientset    kubernetes.Interface

	PodName  string
	FileName string

	genericclioptions.IOStreams
}

// checkDestinationIsDir receives a destination fileSpec and
// determines if the provided destination path exists on the
// pod. If the destination path does not exist or is _not_ a
// directory, an error is returned with the exit code received.
func (o *CopyOptions) checkDestinationIsDir(dest fileSpec) error {
	options := &exec.ExecOptions{
		StreamOptions: exec.StreamOptions{
			IOStreams: genericclioptions.IOStreams{
				Out:    bytes.NewBuffer([]byte{}),
				ErrOut: bytes.NewBuffer([]byte{}),
			},

			Namespace: dest.PodNamespace,
			PodName:   dest.PodName,
		},

		Command:  []string{"test", "-d", dest.File.String()},
		Executor: &exec.DefaultRemoteExecutor{},
	}

	return o.execute(options)
}

//
//func (o *CopyOptions) copyToPod(src, dest fileSpec, options *exec.ExecOptions) error {
//	if _, err := os.Stat(src.File.String()); err != nil {
//		return fmt.Errorf("%s doesn't exist in local filesystem", src.File)
//	}
//	reader, writer := io.Pipe()
//
//	srcFile := src.File.(localPath)
//	destFile := dest.File.(remotePath)
//
//	if err := o.checkDestinationIsDir(dest); err == nil {
//		// If no error, dest.File was found to be a directory.
//		// Copy specified src into it
//		destFile = destFile.Join(srcFile.Base())
//	}
//
//	go func(src localPath, dest remotePath, writer io.WriteCloser) {
//		defer writer.Close()
//		//cmdutil.CheckErr(makeTar(src, dest, writer))
//		makeTar(src, dest, writer)
//	}(srcFile, destFile, writer)
//	var cmdArr []string
//
//	// TODO: Improve error messages by first testing if 'tar' is present in the container?
//	if o.NoPreserve {
//		cmdArr = []string{"tar", "--no-same-permissions", "--no-same-owner", "-xmf", "-"}
//	} else {
//		cmdArr = []string{"tar", "-xmf", "-"}
//	}
//	destFileDir := destFile.Dir().String()
//	if len(destFileDir) > 0 {
//		cmdArr = append(cmdArr, "-C", destFileDir)
//	}
//
//	options.StreamOptions = exec.StreamOptions{
//		IOStreams: genericclioptions.IOStreams{
//			In:     reader,
//			Out:    o.Out,
//			ErrOut: o.ErrOut,
//		},
//		Stdin: true,
//
//		Namespace: dest.PodNamespace,
//		PodName:   dest.PodName,
//	}
//
//	options.Command = cmdArr
//	options.Executor = &exec.DefaultRemoteExecutor{}
//	return o.execute(options)
//}

type TarPipe struct {
	src       fileSpec
	o         *CopyOptions
	reader    *io.PipeReader
	outStream *io.PipeWriter
	bytesRead uint64
	retries   int
}

func newTarPipe(src fileSpec, o *CopyOptions) *TarPipe {
	t := new(TarPipe)
	t.src = src
	t.o = o
	t.initReadFrom(0)
	return t
}

func (t *TarPipe) initReadFrom(n uint64) {
	t.reader, t.outStream = io.Pipe()
	options := &exec.ExecOptions{
		StreamOptions: exec.StreamOptions{
			IOStreams: genericclioptions.IOStreams{
				In:     nil,
				Out:    t.outStream,
				ErrOut: t.o.ErrOut,
			},

			Namespace: t.src.PodNamespace,
			PodName:   t.src.PodName,
		},

		// TODO: Improve error messages by first testing if 'tar' is present in the container?
		Command:  []string{"tar", "cf", "-", t.src.File.String()},
		Executor: &exec.DefaultRemoteExecutor{},
	}
	if t.o.MaxTries != 0 {
		options.Command = []string{"sh", "-c", fmt.Sprintf("tar cf - %s | tail -c+%d", t.src.File, n)}
	}

	go func() {
		defer t.outStream.Close()
		//cmdutil.CheckErr(t.o.execute(options))
		_ = t.o.execute(options) // TODO: 处理错误
	}()
}

func (t *TarPipe) Read(p []byte) (n int, err error) {
	n, err = t.reader.Read(p)
	if err != nil {
		if t.o.MaxTries < 0 || t.retries < t.o.MaxTries {
			t.retries++
			fmt.Printf("Resuming copy at %d bytes, retry %d/%d\n", t.bytesRead, t.retries, t.o.MaxTries)
			t.initReadFrom(t.bytesRead + 1)
			err = nil
		} else {
			fmt.Printf("Dropping out copy after %d retries\n", t.retries)
		}
	} else {
		t.bytesRead += uint64(n)
	}
	return
}

func makeTar(src io.Reader, dest remotePath, writer io.Writer) error {
	// TODO: use compression here?
	tarWriter := tar.NewWriter(writer)
	defer func() { _ = tarWriter.Close() }()

	//destPath := dest.Clean()
	now := time.Now()
	header := &tar.Header{
		Name:       dest.Base().String(),
		Mode:       0644,
		ModTime:    now,
		AccessTime: now,
		ChangeTime: now,
		Size:       19,
		Typeflag:   tar.TypeReg,
	}

	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}
	if _, err := io.Copy(tarWriter, src); err != nil {
		return err
	}
	return nil
}

func (o *CopyOptions) untarAll(prefix string, reader io.Reader) error {
	// TODO: use compression here?
	tarReader := tar.NewReader(reader)
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err != io.EOF {
				return err
			}
			break
		}

		// All the files will start with the prefix, which is the directory where
		// they were located on the pod, we need to strip down that prefix, but
		// if the prefix is missing it means the tar was tempered with.
		// For the case where prefix is empty we need to ensure that the path
		// is not absolute, which also indicates the tar file was tempered with.
		if !strings.HasPrefix(header.Name, prefix) {
			return fmt.Errorf("tar contents corrupted")
		}

		// basic file information
		mode := header.FileInfo().Mode()

		if header.FileInfo().IsDir() {
			continue
		}

		if mode&os.ModeSymlink != 0 {
			fmt.Fprintf(o.IOStreams.ErrOut, "warning: skipping symlink: -> %q\n", header.Linkname)
			continue
		}

		if _, err := io.Copy(o.IOStreams.Out, tarReader); err != nil {
			return err
		}
	}

	return nil
}

func (o *CopyOptions) execute(options *exec.ExecOptions) error {
	if len(options.Namespace) == 0 {
		options.Namespace = o.Namespace
	}

	if len(o.Container) > 0 {
		options.ContainerName = o.Container
	}

	options.Config = o.ClientConfig
	options.PodClient = o.Clientset.CoreV1()

	if err := options.Run(); err != nil {
		return err
	}
	return nil
}

func (o *CopyOptions) CopyFromPod() error {
	srcFile := remotePath{o.FileName}
	src := fileSpec{
		PodName:      o.PodName,
		PodNamespace: o.Namespace,
		File:         srcFile,
	}
	if err := o.checkDestinationIsDir(src); err == nil {
		return fmt.Errorf("can not download directory: %v", o.FileName)
	}
	reader := newTarPipe(src, o)
	prefix := stripPathShortcuts(srcFile.StripSlashes().Clean().String())
	return o.untarAll(prefix, reader)
}

func (o *CopyOptions) CopyToPod() error {
	destFile := remotePath{o.FileName}
	dest := fileSpec{
		PodName:      o.PodName,
		PodNamespace: o.Namespace,
		File:         destFile,
	}
	reader, writer := io.Pipe()

	if err := o.checkDestinationIsDir(dest); err == nil {
		return fmt.Errorf("can not upload directory: %v", o.FileName)
	}

	go func(src io.Reader, dest remotePath, writer io.WriteCloser) {
		defer func() { _ = writer.Close() }()
		if err := makeTar(src, dest, writer); err != nil { // TODO: 错误处理
			fmt.Println("makeTar error: ", err)
		}
	}(o.IOStreams.In, destFile, writer)
	var cmdArr []string

	// TODO: Improve error messages by first testing if 'tar' is present in the container?
	if o.NoPreserve {
		cmdArr = []string{"tar", "--no-same-permissions", "--no-same-owner", "-xmf", "-"}
	} else {
		cmdArr = []string{"tar", "-xmf", "-"}
	}
	destFileDir := destFile.Dir().String()
	if len(destFileDir) > 0 {
		cmdArr = append(cmdArr, "-C", destFileDir)
	}
	options := &exec.ExecOptions{}
	options.StreamOptions = exec.StreamOptions{
		IOStreams: genericclioptions.IOStreams{
			In:     reader,
			Out:    o.Out,
			ErrOut: o.ErrOut,
		},
		Stdin: true,

		Namespace: dest.PodNamespace,
		PodName:   dest.PodName,
	}

	options.Command = cmdArr
	options.Executor = &exec.DefaultRemoteExecutor{}
	err := o.execute(options)
	//fmt.Println(ioutil.ReadAll(options.IOStreams.ErrOut))
	return err
}
