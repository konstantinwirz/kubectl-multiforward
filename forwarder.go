package main

import (
	"bytes"
	"fmt"
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

func (f Forwarder) Start(wg *sync.WaitGroup, poder Poder, resultsChan chan<- ForwardResult, stopChan <-chan struct{}) error {
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

	forwarder, err := portforward.New(dialer, poder.Ports(), stopChan, readyChan, out, errOut)
	if err != nil {
		return fmt.Errorf("error creating port forwarder: %w", err)
	}

	go func() {
		// Kubernetes will close this channel when it has something to tell us
		<-readyChan

		if len(errOut.String()) != 0 {
			fmt.Printf("[%s/%s] [stderr]: %s\n", poder.Namespace(), pod, errOut)
		}

		if len(out.String()) != 0 {
			fmt.Printf("[%s/%s] [stdout]: %s\n", poder.Namespace(), pod, strings.Replace(out.String(), "\n", "; ", -1))
		}

	}()

	go func() {
		defer wg.Done()

		fmt.Printf("[%s/%s] port forwarding established\n", poder.Namespace(), pod)

		if err = forwarder.ForwardPorts(); err != nil {
			fmt.Printf("[%s/%s] error forwarding ports: %s\n", poder.Namespace(), pod, err.Error())
			resultsChan <- NewForwardResultWithError(poder, err)
		}
		resultsChan <- NewForwardResult(poder)
	}()

	return nil
}

func (f Forwarder) StartMany(poders []Poder, stopChan <-chan struct{}) (<-chan struct{}, error) {
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
					// restart the forwarder
					err := f.Start(&wg, result.Source, resultsChan, targetStopChan)
					if err != nil {
						fmt.Printf("error restarting forwarder, try it again in a loop -> %s\n", err.Error())

						go func() {
							t := time.NewTicker(5 * time.Second)
							for {
								select {
								case <-stopChan:
									fmt.Println("received stop signal, no more attempts to restart forwarder")
									return
								case <-t.C:
									if err := f.Start(&wg, result.Source, resultsChan, targetStopChan); err == nil {
										t.Stop()
										wg.Add(1)
										return
									}
									// just do it again until err == nil
								}
							}
						}()

					} else {
						wg.Add(1)
					}
				}
			case <-stopChan:
				fmt.Printf("received stop signal, stopping all forwarders...")
				close(targetStopChan)
				wg.Wait()
				fmt.Printf("DONE\n")
				break loop
			}
		}
	}()

	for _, poder := range poders {
		wg.Add(1)
		if err := f.Start(&wg, poder, resultsChan, targetStopChan); err != nil {
			return doneChan, fmt.Errorf("error starting forwarder: %w", err)
		}
	}

	return doneChan, nil
}
