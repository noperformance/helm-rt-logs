# Helm real time logs Plugin

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Report Card](https://goreportcard.com/badge/github.com/noperformance/helm-rt-logs)](https://goreportcard.com/report/github.com/noperformance/helm-rt-logs)
[![CircleCI](https://dl.circleci.com/status-badge/img/circleci/L6LqkTDTpv1YyfotNqY4bH/9yV8FQC1uYaNy7ug5dzyxx/tree/main.svg?style=svg&circle-token=73e2fd2d2d2f01cd03a1d832f58a56ec596026f0)](https://dl.circleci.com/status-badge/redirect/circleci/L6LqkTDTpv1YyfotNqY4bH/9yV8FQC1uYaNy7ug5dzyxx/tree/main)
[![Release](https://img.shields.io/github/release/noperformance/helm-rt-logs.svg?style=flat-square)](https://github.com/noperformance/helm-rt-logs/releases/latest)

## Overview:

rt-logs is a Helm plugin designed for CI/CD pipelines, enabling real-time log collection from Kubernetes resources without direct cluster access. It's tailored for scenarios requiring post-release log analysis in restricted-access environments.

## Prerequisite

- Helm client with `rt-logs` plugin installed on the same system
- Access to the cluster(s) that Helm manages. This access is similar to `kubectl` access using [kubeconfig files](https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig/).
  The `--kubeconfig`, `--kube-context` and `--namespace` flags can be used to set the kubeconfig path, kube context and namespace context to override the environment configuration.

## Install

Based on the version in `plugin.yaml`, release binary will be downloaded from GitHub:

```console
$ helm plugin install https://github.com/noperformance/helm-rt-logs
Downloading and installing helm-rt-logs v0.1.0 ...
https://github.com/noperformance/helm-rt-logs/releases/download/v0.1.0/helm-rt-logs_0.1.0_darwin_amd64.tar.gz
Installed plugin: rt-logs
```

### For Windows (using WSL)

Helm's plugin install hook system relies on `/bin/sh`, regardless of the operating system present. Windows users can work around this by using Helm under [WSL](https://docs.microsoft.com/en-us/windows/wsl/install-win10).
```
$ wget https://get.helm.sh/helm-v3.0.0-linux-amd64.tar.gz
$ tar xzf helm-v3.0.0-linux-amd64.tar.gz
$ ./linux-amd64/helm plugin install https://github.com/noperformance/helm-rt-logs
```

## Usage

```console
$ helm rt-logs RELEASE [flags]

		tail logs of a release

Usage:
  rtlogs [flags] RELEASE

Flags:
  -h, --help                    help for rtlogs
      --stop-string string      string to stop the logs
      --stop-timeout int        timeout to stop the logs, in Seconds!
  -o, --only-failed             tail logs only from failed pods
  -s, --time-since int          time since to start the logs
  -d, --debug                   enables debug messages
```

no old helm! 
