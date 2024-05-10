package main

import (
	"fmt"
	"os"
)

type Severity int

const (
	SeverityTrace Severity = iota
	SeverityDebug
	SeverityWarning
	SeverityError
	SeverityInfo
)

func (s Severity) String() string {
	switch s {
	case SeverityTrace:
		return "TRACE"
	case SeverityDebug:
		return "DEBUG"
	case SeverityWarning:
		return "WARNING"
	case SeverityError:
		return "ERROR"
	case SeverityInfo:
		return "INFO"
	default:
		return "UNKNOWN"
	}
}

type Report struct {
	Severity Severity
	Message  string
}

func (r Report) String() string {
	return r.Message
}

func NewReport(severity Severity, poder Poder, format string, a ...any) Report {
	prefix := fmt.Sprintf("[%s] ", severity)
	if poder != nil {
		prefix = fmt.Sprintf("[%s] [%s] ", severity, poder)
	}

	return Report{
		Severity: severity,
		Message:  prefix + fmt.Sprintf(format, a...),
	}
}

const Reset = "\033[0m"
const Red = "\033[31m"
const Green = "\033[32m"
const Yellow = "\033[33m"
const Cyan = "\033[36m"

func (r Report) Dump() {
	switch r.Severity {
	case SeverityInfo:
		fmt.Println(Green + r.Message + Reset)
	case SeverityWarning:
		fmt.Println(Yellow + r.Message + Reset)
	case SeverityError:
		fmt.Fprintln(os.Stderr, Red+r.Message+Reset)
	default:
		fmt.Println(Cyan + r.Message + Reset)
	}
}
