package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var (
	// will be replaced by goreleaser (using ldflags)
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	var namespace string
	var kubeConfigPath string
	var severity string

	var rootCmd = &cobra.Command{
		Use:   "kubectl-multiforward [flags] resource1 resource2 ... resourceN",
		Short: "Port-Forward multiple k8s resources simultaneously",
		Long: `
Port-Forward multiple k8s resources simultaneously.

A resource is specified as [namespace/]type/name:localPort:remotePort.

Following resource types can be forwarded:
 - pods
 - deployments
 - services
`,
		Version: fmt.Sprintf("%s (commit: %s, date: %s)", version, commit, date),
		Args:    cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			forward(args, namespace, kubeConfigPath, severity)
		},
	}

	flags := rootCmd.Flags()
	flags.StringVarP(&namespace, "namespace", "n", "", "k8s namespace which will be used for all resources (if not set otherwise)")
	flags.StringVarP(&kubeConfigPath, "kubeconfig", "k", filepath.Join(homedir.HomeDir(), ".kube", "config"), "path to kubeconfig file")
	flags.StringVarP(&severity, "severity", "s", "info", "log severity (trace, debug, info, warning, error)")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing root command: %s\n", err.Error())
		os.Exit(1)
	}
}

func forward(resources []string, namespace string, kubeConfigPath string, severity string) {
	if len(resources) == 0 {
		// cannot happen
		panic("no resources specified")
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

	currentSeverity, err = SeverityFromString(severity)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error recognizing severity: %s\n", err.Error())
		os.Exit(1)
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
	reportChan := make(chan Report, len(poder)*10)
	doneChan, err := forwarder.Forward(poder, stopChan, reportChan)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting forwarder: %s\n", err.Error())
		os.Exit(1)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-c:
			NewReport(SeverityInfo, nil, "sending stop signal to all forwarders...").Dump()
			close(stopChan)
		case report := <-reportChan:
			report.Dump()
		case <-doneChan:
			NewReport(SeverityInfo, nil, "all forwarders finished, quit...").Dump()
			os.Exit(0)
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
