# kubectl-multiforward
[![Build](https://github.com/konstantinwirz/kubectl-multiforward/actions/workflows/build.yaml/badge.svg)](https://github.com/konstantinwirz/kubectl-multiforward/actions/workflows/build.yaml) [![Lint](https://github.com/konstantinwirz/kubectl-multiforward/actions/workflows/lint.yaml/badge.svg)](https://github.com/konstantinwirz/kubectl-multiforward/actions/workflows/lint.yaml)
 
kubectl plugin to maintain port forwards with multiple resources

## Installation

```shell
$ go install github.com/konstantinwirz/kubectl-multiforward@latest
```

## Usage

```shell
$ kubectl multiforward longhorn-system/service/longhorn-frontend:8080:8000 pihole/service/pihole-web:8081:80
```