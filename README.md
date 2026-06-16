# helm rt-logs

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Report Card](https://goreportcard.com/badge/github.com/noperformance/helm-rt-logs)](https://goreportcard.com/report/github.com/noperformance/helm-rt-logs)
[![CircleCI](https://dl.circleci.com/status-badge/img/circleci/L6LqkTDTpv1YyfotNqY4bH/9yV8FQC1uYaNy7ug5dzyxx/tree/main.svg?style=svg&circle-token=73e2fd2d2d2f01cd03a1d832f58a56ec596026f0)](https://dl.circleci.com/status-badge/redirect/circleci/L6LqkTDTpv1YyfotNqY4bH/9yV8FQC1uYaNy7ug5dzyxx/tree/main)
[![Release](https://img.shields.io/github/release/noperformance/helm-rt-logs.svg?style=flat-square)](https://github.com/noperformance/helm-rt-logs/releases/latest)

Stream logs from all pods of a Helm release in real time. Built for CI/CD: run it
after a release to watch every Deployment, StatefulSet and DaemonSet pod, and stop
on a timeout, a marker line, or an interrupt — so the pipeline never hangs.

## Install

```console
$ helm plugin install https://github.com/noperformance/helm-rt-logs
Installed plugin: rt-logs
```

The binary version comes from `plugin.yaml`. On Windows use [WSL](https://docs.microsoft.com/en-us/windows/wsl/install-win10)
(Helm's install hooks need `/bin/sh`). Requires Helm 3 and `kubectl`-style cluster
access via a [kubeconfig](https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig/).

## Usage

```console
$ helm rt-logs RELEASE [helm flags] -- [plugin flags]
```

Helm's own flags (`-n/--namespace`, `--kube-context`, `--kubeconfig`, `--debug`) go
**before** `--`; Helm reads them and forwards them to the plugin via `HELM_*` env
vars. Plugin flags go **after** `--`, where Helm passes them through untouched.

> The `--` separator avoids any clash. In particular `--debug` belongs to Helm —
> enable the plugin's own debug with `-d` after `--`.

### Plugin flags

| Flag | Short | Description |
|------|-------|-------------|
| `--container` | `-c` | tail only this container (default: all) |
| `--only-failed` | `-o` | tail only pods that are not Running |
| `--stop-string` | | stop once this substring appears in a log line |
| `--stop-timeout` | | stop after N seconds |
| `--time-since` | `-s` | show logs newer than N seconds |
| `--tail` | | last N lines per container (`-1` = all) |
| `--timestamps` | | prefix each line with a timestamp |
| `--debug` | `-d` | enable debug output |

Tailing stops when every stream ends, the stop timeout fires, the stop string is
matched, or the process gets `SIGINT`/`SIGTERM` (Ctrl-C).

## Examples

Tail everything in the release:
```console
$ helm rt-logs my-release
```

Pick namespace and context (Helm flags, before `--`):
```console
$ helm -n production --kube-context staging rt-logs my-release
```

Plugin flags after `--` — last 60s, stop after 2 min:
```console
$ helm rt-logs my-release -- --time-since 60 --stop-timeout 120
```

Wait for a marker line, then exit (handy in CI):
```console
$ helm rt-logs my-release -- --stop-string "Server started on :8080"
```

Only failing pods, single container, plugin debug:
```console
$ helm -n prod rt-logs my-release -- -o -c app -d
```

CI gate: wait for a migration, cap the wait so the job can't hang:
```console
$ helm -n prod --kube-context ci-cluster rt-logs my-release \
    -- --stop-string "Migration complete" --stop-timeout 300 -c migrate
```

### Alongside `helm install` / `helm upgrade`

The release must already exist, so run `rt-logs` right after the deploy. Watch the
rollout with a time cap so the step can't hang:
```console
$ helm upgrade --install my-release ./chart -n prod
$ helm rt-logs my-release -n prod -- --stop-timeout 180
```

Deploy, then watch until a readiness marker appears (or the timeout fires):
```console
$ helm upgrade --install my-release ./chart -n prod
$ helm rt-logs my-release -n prod -- --stop-string "listening on :8080" --stop-timeout 120
```

Inspect a bad rollout — only non-Running pods, last 50 lines each, with timestamps:
```console
$ helm upgrade --install my-release ./chart -n prod
$ helm rt-logs my-release -n prod -- -o --tail 50 --timestamps --stop-timeout 60
```

> `--wait` on `helm upgrade` blocks until pods are Ready, which hides early startup
> logs. Drop `--wait` (as above) when you want `rt-logs` to stream the rollout live.

## Expected output

Line format: `[type/name][pod][container][phase] log line`.

```text
[deployment/web][pod=web-7c9f8d4b6-abcde][container=app][phase=Running] 2026/06/16 12:01:03 INFO server listening on :8080
[statefulset/db][pod=db-0][container=postgres][phase=Running] 2026-06-16 12:01:05 UTC LOG:  database system is ready to accept connections
```

A pod still scheduling (re-checked every 5s, never loops forever):
```text
[Pod web-7c9f8d4b6-abcde] still pending
[deployment/web][pod=web-7c9f8d4b6-abcde][container=app][phase=Running] 2026/06/16 12:01:10 INFO booting
```

A failed pod (with `-o`):
```text
[Pod api-5d6c-xfz2] phase=Failed message="back-off restarting failed container" reason="CrashLoopBackOff"
[deployment/api][pod=api-5d6c-xfz2][container=api][phase=Failed] panic: cannot bind :8080: address already in use
```

Nothing to tail:
```text
No pods to tail logs from found (with -o all pods may be Running, otherwise the release has no pods).
```

## Roadmap

- [ ] Watch for pods created after start (rollout-aware streaming)
- [ ] Include Jobs / CronJobs / bare Pods in discovery
