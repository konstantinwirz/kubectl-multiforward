package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type arrayFlags []string

// String implements flag.Value
func (af *arrayFlags) String() string {
	return strings.Join(*af, ", ")
}

// Set implements flag.Value
func (af *arrayFlags) Set(value string) error {
	*af = append(*af, value)
	return nil
}

var doc = `
TODO: make it right!
Forward one or more local ports to a pod.

 Use resource type/name such as deployment/mydeployment to select a pod. Resource type defaults to
'pod' if omitted.
`

func init() {

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "%s\n", doc)
		flag.PrintDefaults()
	}
}

func main() {
	var resources arrayFlags
	var namespace string
	var kubeConfigPath string
	flag.Var(&resources, "resource", "resource [namespace/]type/name:localPort:remotePort")
	flag.StringVar(&namespace, "namespace", "", "k8s namespace for all resources")
	flag.StringVar(&kubeConfigPath, "kubeconfig", filepath.Join(homedir.HomeDir(), ".kube", "config"), "path to kubeconfig file")

	flag.Parse()

	if len(resources) == 0 {
		fmt.Fprintf(os.Stderr, "at least one resource must be specified\n")
		os.Exit(1)
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	if strings.TrimSpace(namespace) == "" {
		namespace, err = getDefaultNamespaceFromCtx(kubeConfigPath)
		if err != nil {
			fmt.Printf("couldn't determine default namespace, using 'default': %s\n", err.Error())
			namespace = "default"
		}
	}

	var poder []Poder
	for _, s := range resources {
		r, err := ParseResource(s)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing resource: %s\n", err.Error())
			os.Exit(1)
		}
		if r.Namespace == "" {
			r.Namespace = namespace
		}
		poder = append(poder, NewPoder(config, r))
	}

	forwarder := NewForwarder(config)

	stopChan := make(chan struct{})
	doneChan, err := forwarder.StartMany(poder, stopChan)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting forwarder: %s\n", err.Error())
		os.Exit(1)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

mainloop:
	for {
		select {
		case <-c:
			fmt.Println("sending stop signal to forwarder...")
			close(stopChan)
		case <-doneChan:
			fmt.Printf("all forwarders stopped\n")
			break mainloop
		}
	}
}

func getDefaultNamespaceFromCtx(kubeConfigPath string) (string, error) {
	config, err := clientcmd.LoadFromFile(kubeConfigPath)
	if err != nil {
		return "", fmt.Errorf("failed to load kubeconfig: %s", err.Error())
	}

	currentCtx, ok := config.Contexts[config.CurrentContext]
	if !ok {
		return "", fmt.Errorf("current context not found")
	}

	return currentCtx.Namespace, nil
}
