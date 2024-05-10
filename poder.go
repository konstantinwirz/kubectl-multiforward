package main

import (
	"context"
	"fmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"math/rand"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Poder interface {
	fmt.Stringer
	Pod() (string, error)
	Namespace() string
	Ports() []string
}

type podPoder struct {
	k8sConfig      *rest.Config
	namespace, pod string
	ports          []string
}

var _ Poder = &podPoder{}

func (p podPoder) Namespace() string {
	return p.namespace
}

func (p podPoder) Pod() (string, error) {
	clientset, err := kubernetes.NewForConfig(p.k8sConfig)
	if err != nil {
		return "", fmt.Errorf("error creating k8s clientset: %s", err)
	}

	_, err = clientset.CoreV1().Pods(p.Namespace()).Get(context.Background(), p.pod, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("error getting pod: %s", err)
	}

	return p.pod, nil
}

func (p podPoder) Ports() []string {
	return p.ports
}

func (p podPoder) String() string {
	return fmt.Sprintf("%s/%s", p.namespace, p.pod)
}

func NewPoder(config *rest.Config, resource Resource) Poder {
	switch resource.Type {
	case Pod:
		return podPoder{
			k8sConfig: config,
			namespace: resource.Namespace,
			pod:       resource.Name,
			ports:     []string{resource.Ports},
		}

	case Service:
		return servicePoder{
			namespace: resource.Namespace,
			service:   resource.Name,
			ports:     []string{resource.Ports},
			k8sConfig: config,
		}

	case Deployment:
		return deploymentPoder{
			namespace:  resource.Namespace,
			deployment: resource.Name,
			ports:      []string{resource.Ports},
			k8sConfig:  config,
		}
	default:
		panic("Unknown resource type")
	}
}

type servicePoder struct {
	k8sConfig          *rest.Config
	namespace, service string
	ports              []string
}

var _ Poder = &servicePoder{}

func (p servicePoder) Namespace() string {
	return p.namespace
}

func (p servicePoder) Ports() []string {
	return p.ports
}

func (p servicePoder) Pod() (string, error) {
	return PickRandomPod(p.k8sConfig, p.namespace, p.service, fetchPodsForService)
}

func (p servicePoder) String() string {
	return fmt.Sprintf("%s/%s", p.namespace, p.service)
}

type deploymentPoder struct {
	k8sConfig             *rest.Config
	namespace, deployment string
	ports                 []string
}

var _ Poder = &deploymentPoder{}

func (p deploymentPoder) Pod() (string, error) {
	return PickRandomPod(p.k8sConfig, p.namespace, p.deployment, fetchPodsForDeployment)
}

func (p deploymentPoder) Namespace() string {
	return p.namespace
}

func (p deploymentPoder) Ports() []string {
	return p.ports
}

func (p deploymentPoder) String() string {
	return fmt.Sprintf("%s/%s", p.namespace, p.deployment)
}

// fetchPodsForService gets all pods for a k8s service
func fetchPodsForService(config *rest.Config, namespace, service string) ([]string, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating k8s clientset: %w", err)
	}

	endpoints, err := clientset.CoreV1().Endpoints(namespace).Get(context.Background(), service, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting endpoints: %w", err)
	}

	var pods []string
	for _, subset := range endpoints.Subsets {
		for _, addr := range subset.Addresses {
			if addr.TargetRef != nil {
				pods = append(pods, addr.TargetRef.Name)
			}
		}
	}

	return pods, nil
}

func fetchPodsForDeployment(config *rest.Config, namespace, deployment string) ([]string, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating k8s clientset -> %s", err)
	}

	podList, err := clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting deployment -> %s", err)
	}

	var pods []string
	for _, podItem := range podList.Items {
		for _, ownerRef := range podItem.OwnerReferences {
			if ownerRef.Kind == "ReplicaSet" {
				replicaSet, err := clientset.AppsV1().ReplicaSets(namespace).Get(context.Background(), ownerRef.Name, metav1.GetOptions{})
				if err != nil {
					return nil, fmt.Errorf("error getting replica set -> %s", err)
				}
				for _, rsOwnerRef := range replicaSet.OwnerReferences {
					if rsOwnerRef.Kind == "Deployment" && rsOwnerRef.Name == deployment {
						pods = append(pods, podItem.Name)
					}
				}
			}
		}
	}

	if len(pods) == 0 {
		return nil, fmt.Errorf("no pods found for deployment")
	}

	return pods, nil
}

func PickRandomPod(config *rest.Config, namespace, svc string, fetchPodsFunc func(*rest.Config, string, string) ([]string, error)) (string, error) {
	pods, err := fetchPodsFunc(config, namespace, svc)
	if err != nil {
		return "", fmt.Errorf("error fetching pod names: %w", err)
	}

	if len(pods) == 0 {
		return "", fmt.Errorf("no pods found")
	}

	return pickRandom(pods), nil
}

func pickRandom[T any](slice []T) T {
	if len(slice) == 0 {
		panic("Empty slice")
	}

	var idx = 0
	if len(slice) > 1 {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		idx = r.Intn(len(slice))
	}

	return slice[idx]
}
