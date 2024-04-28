package main

import (
	"fmt"
	"regexp"
)

type ResourceType string

var (
	Undefined  ResourceType = "undefined"
	Pod        ResourceType = "pod"
	Service    ResourceType = "service"
	Deployment ResourceType = "deployment"
)

func ResourceTypeFromString(s string) ResourceType {
	switch s {
	case "pod":
		return Pod
	case "service":
		return Service
	case "deployment":
		return Deployment
	default:
		return Undefined
	}
}

// Resource represents a resource which can be port forwarded
//
// types of resources that can be forwarded:
// - [namespace/]service/name:port:port
// - [namespace/]deployment/name:port:port
// - [namespace/]pod/name:port:port
type Resource struct {
	Type      ResourceType
	Namespace string
	Name      string
	Ports     string
}

// ParseResource parses given string into a Resource
func ParseResource(s string) (Resource, error) {
	re := regexp.MustCompile(`((\S+)/)?(service|pod|deployment)/(\S+):(\d+:\d+)`)
	matches := re.FindStringSubmatch(s)
	if len(matches) < 3 {
		return Resource{}, fmt.Errorf("invalid resource format: %s", s)
	}

	// we have at least ports, name and type
	ports := matches[len(matches)-1]
	name := matches[len(matches)-2]
	t := matches[len(matches)-3]
	namespace := ""
	if len(matches) > 3 {
		namespace = matches[len(matches)-4]
	}

	return Resource{
		Type:      ResourceTypeFromString(t),
		Namespace: namespace,
		Name:      name,
		Ports:     ports,
	}, nil
}
