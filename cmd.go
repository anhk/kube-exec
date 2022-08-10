package exec

import (
	"os"

	"github.com/anhk/exec/exec"
	"github.com/anhk/exec/scheme"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Shell struct {
	client *kubernetes.Clientset
	config *restclient.Config

	Namespace string
	PodName   string
	Command   string
	Args      []string
}

type Client struct {
	client *kubernetes.Clientset
	config *restclient.Config
}

func NewClient(cfgPath string) (*Client, error) {
	cfg, err := clientcmd.BuildConfigFromFlags("", cfgPath)
	if err != nil {
		return nil, err
	}
	cfg.TLSClientConfig.ServerName = "192.168.49.2" // for test <minikube>
	cfg.GroupVersion = &corev1.SchemeGroupVersion
	cfg.NegotiatedSerializer = scheme.Codecs
	cfg.APIPath = "/api"

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &Client{client: client, config: cfg}, nil
}

func (cli *Client) Shell(namespace, podName, command string, args ...string) *Shell {
	return &Shell{
		client: cli.client,
		config: cli.config,

		Namespace: namespace,
		PodName:   podName,
		Command:   command,
		Args:      args,
	}
}

func (s *Shell) Run() error {
	options := &exec.ExecOptions{
		StreamOptions: exec.StreamOptions{
			IOStreams: genericclioptions.IOStreams{
				In:     os.Stdin,
				Out:    os.Stdout,
				ErrOut: os.Stderr,
			},
			Stdin:     true,
			TTY:       true,
			Namespace: s.Namespace,
			PodName:   s.PodName,
		},
		Config:    s.config,
		PodClient: s.client.CoreV1(),
		Command:   []string{s.Command},
		Executor:  &exec.DefaultRemoteExecutor{},
	}
	options.Command = append(options.Command, s.Args...)

	return options.Run()
}
