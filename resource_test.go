package main

import (
	"fmt"
	"reflect"
	"testing"
)

func TestResourceTypeFromString(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want ResourceType
	}{
		{
			name: "pod",
			s:    "pod",
			want: Pod,
		},
		{
			name: "service",
			s:    "service",
			want: Service,
		},
		{
			name: "deployment",
			s:    "deployment",
			want: Deployment,
		},
		{
			name: "empty string",
			s:    "",
			want: Undefined,
		},
		{
			name: "unknown resource type",
			s:    "foobar",
			want: Undefined,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResourceTypeFromString(tt.s)
			if got != tt.want {
				t.Errorf("ResourceTypeFromString() = %v, want %v", got, tt.want)
			}
		})
	}

}

func TestParseResource(t *testing.T) {
	tests := []struct {
		name    string
		s       string
		want    Resource
		wantErr error
	}{
		{
			name: "valid pod without namespace",
			s:    "pod/foo:8080:8080",
			want: Resource{
				Type:      Pod,
				Namespace: "",
				Name:      "foo",
				Ports:     "8080:8080",
			},
			wantErr: nil,
		},
		{
			name: "valid pod with namespace",
			s:    "ns/pod/foo:8080:8080",
			want: Resource{
				Type:      Pod,
				Namespace: "ns",
				Name:      "foo",
				Ports:     "8080:8080",
			},
			wantErr: nil,
		},
		{
			name: "valid service without namespace",
			s:    "service/foo:8080:8080",
			want: Resource{
				Type:      Service,
				Namespace: "",
				Name:      "foo",
				Ports:     "8080:8080",
			},
			wantErr: nil,
		},
		{
			name: "valid service with namespace",
			s:    "ns/service/foo:8080:8080",
			want: Resource{
				Type:      Service,
				Namespace: "ns",
				Name:      "foo",
				Ports:     "8080:8080",
			},
			wantErr: nil,
		},
		{
			name: "valid deployment without namespace",
			s:    "deployment/foo:8080:8080",
			want: Resource{
				Type:      Deployment,
				Namespace: "",
				Name:      "foo",
				Ports:     "8080:8080",
			},
			wantErr: nil,
		},
		{
			name: "valid deployment wit namespace",
			s:    "ns/deployment/foo:8080:8080",
			want: Resource{
				Type:      Deployment,
				Namespace: "ns",
				Name:      "foo",
				Ports:     "8080:8080",
			},
			wantErr: nil,
		},
		{
			name:    "unknown resource type",
			s:       "foo/bar:8080:8080",
			want:    Resource{},
			wantErr: fmt.Errorf("invalid resource format: foo/bar:8080:8080"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseResource(tt.s)
			if tt.wantErr == nil && err != nil {
				t.Fatalf("ParseResource() didn't expect an error, got: %v", err)
			}

			if tt.wantErr != nil && err == nil {
				t.Fatalf("ParseResource() expected an error, got nil")
			}

			if tt.wantErr != nil && !reflect.DeepEqual(tt.wantErr, err) {
				t.Fatalf("ParseResource() expected an error = %v, got: %v", tt.wantErr, err)
			}

			if got != tt.want {
				t.Fatalf("ParseResource() = %v, want %v", got, tt.want)
			}
		})
	}
}
