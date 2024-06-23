package main

import (
	"fmt"
	"reflect"
	"testing"
)

func TestSeverityFromString(t *testing.T) {
	tests := []struct {
		name    string
		s       string
		want    Severity
		wantErr error
	}{
		{
			name:    "trace",
			s:       "trace",
			want:    SeverityTrace,
			wantErr: nil,
		},
		{
			name:    "trace in different case",
			s:       "TRAce",
			want:    SeverityTrace,
			wantErr: nil,
		},
		{
			name:    "info",
			s:       "INFO",
			want:    SeverityInfo,
			wantErr: nil,
		},
		{
			name:    "debug",
			s:       "DEBUG",
			want:    SeverityDebug,
			wantErr: nil,
		},
		{
			name:    "warning",
			s:       "warning",
			want:    SeverityWarning,
			wantErr: nil,
		},
		{
			name:    "error",
			s:       "error",
			want:    SeverityError,
			wantErr: nil,
		},
		{
			name:    "invalid severity",
			s:       "foobar",
			want:    SeverityInfo,
			wantErr: fmt.Errorf("unknown severity: foobar"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SeverityFromString(tt.s)

			if !reflect.DeepEqual(tt.wantErr, err) {
				t.Errorf("got error = %v, want %v", err, tt.wantErr)
			}

			if got != tt.want {
				t.Errorf("got severity = %v, want %v", got, tt.want)
			}
		})
	}
}
