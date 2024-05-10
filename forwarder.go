package main

import (
	"bytes"
	"fmt"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Forwarder struct {
	k8sConfig *rest.Config
}

func NewForwarder(k8sConfig *rest.Config) Forwarder {
	return Forwarder{
		k8sConfig: k8sConfig,
	}
}

type ForwardResult struct {
	Source Poder
	Err    error
}

func (fr ForwardResult) IsError() bool {
	return fr.Err != nil
}

func NewForwardResult(poder Poder) ForwardResult {
	return ForwardResult{
		Source: poder,
	}
}

func NewForwardResultWithError(poder Poder, err error) ForwardResult {
	return ForwardResult{
		Source: poder,
		Err:    err,
	}
}

// forwardSingle establishes a single port forwarding connection for a given Poder.
func (f Forwarder) forwardSingle(
	wg *sync.WaitGroup,
	poder Poder,
	resultsChan chan<- ForwardResult,
	stopChan <-chan struct{},
	reportChan chan<- Report,
) error {
	pod, err := poder.Pod()
	if err != nil {
		return fmt.Errorf("couldn't establish port forwarding -> %s", err)
	}

	roundTripper, upgrader, err := spdy.RoundTripperFor(f.k8sConfig)
	if err != nil {
		return fmt.Errorf("error building round tripper: %w", err)
	}

	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", poder.Namespace(), pod)

	serverURL, err := url.Parse(f.k8sConfig.Host + path)
	if err != nil {
		return fmt.Errorf("error parsing k8s server URL '%s'  -> %s", f.k8sConfig.Host, err)
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, http.MethodPost, serverURL)

	readyChan := make(chan struct{}, 1)
	out, errOut := new(bytes.Buffer), new(bytes.Buffer)

	oldRuntimeErrorHandlers := runtime.ErrorHandlers
	runtime.ErrorHandlers = []func(error){
		func(err error) {
			reportChan <- NewReport(SeverityError, poder, err.Error())
		},
	}
	if len(oldRuntimeErrorHandlers) >= 2 {
		runtime.ErrorHandlers = append(runtime.ErrorHandlers, oldRuntimeErrorHandlers[1])
	}

	forwarder, err := portforward.New(dialer, poder.Ports(), stopChan, readyChan, out, errOut)
	if err != nil {
		return fmt.Errorf("error creating port forwarder: %w", err)
	}

	go func() {
		// Kubernetes will close this channel when it has something to tell us
		<-readyChan

		if len(errOut.String()) != 0 {
			reportChan <- NewReport(SeverityError, poder, strings.TrimSpace(strings.ReplaceAll(out.String(), "\n", "; ")))
		}

		if len(out.String()) != 0 {
			reportChan <- NewReport(SeverityInfo, poder, strings.TrimSpace(strings.ReplaceAll(out.String(), "\n", "; ")))
		}
	}()

	go func() {
		defer wg.Done()

		reportChan <- NewReport(SeverityDebug, poder, "establishing port forwarding for %s ...", pod)

		if err = forwarder.ForwardPorts(); err != nil {
			reportChan <- NewReport(SeverityError, poder, "error forwarding ports: %s", err.Error())
			resultsChan <- NewForwardResultWithError(poder, err)
		}
		resultsChan <- NewForwardResult(poder)
	}()

	return nil
}

func (f Forwarder) forwardSingleInALoop(
	wg *sync.WaitGroup,
	poder Poder,
	resultsChan chan<- ForwardResult,
	stopChan <-chan struct{},
	reportChan chan<- Report,
) {
	t := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-stopChan:
			reportChan <- NewReport(SeverityInfo, poder, "received stop signal, no more attempts to restart forwarder")
			return
		case <-t.C:
			reportChan <- NewReport(SeverityTrace, poder, "trying to restart forwarder...")
			if err := f.forwardSingle(wg, poder, resultsChan, stopChan, reportChan); err == nil {
				t.Stop()
				wg.Add(1)
				reportChan <- NewReport(SeverityInfo, poder, "restarted forwarder...")
				return
			}
			// just do it again until err == nil
		}
	}
}

// Forward establishes port forwarding for all given Poder instances.
func (f Forwarder) Forward(
	poders []Poder,
	stopChan <-chan struct{},
	reportChan chan<- Report,
) (<-chan struct{}, error) {
	resultsChan := make(chan ForwardResult, len(poders))
	targetStopChan := make(chan struct{})
	doneChan := make(chan struct{})
	wg := sync.WaitGroup{}

	go func() {
		defer close(resultsChan)
		defer close(doneChan)

	loop:
		for {
			select {
			case result := <-resultsChan:
				if result.IsError() {
					go f.forwardSingleInALoop(&wg, result.Source, resultsChan, targetStopChan, reportChan)
				}
			case <-stopChan:
				reportChan <- NewReport(SeverityInfo, nil, "received stop signal, stopping all forwarders...")
				close(targetStopChan)
				wg.Wait()
				reportChan <- NewReport(SeverityInfo, nil, "all forwarders stopped")
				break loop
			}
		}
	}()

	for _, poder := range poders {
		wg.Add(1)
		if err := f.forwardSingle(&wg, poder, resultsChan, targetStopChan, reportChan); err != nil {
			return doneChan, fmt.Errorf("error starting forwarder: %w", err)
		}
	}

	return doneChan, nil
}
