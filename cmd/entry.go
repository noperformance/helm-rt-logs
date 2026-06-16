package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"helm-rt-logs/pkg/collector"
	"helm-rt-logs/pkg/kubeclient"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
)

// rtLogsCmd holds the parsed flags and runtime dependencies for the rtlogs command.
type rtLogsCmd struct {
	release     string // release name
	stopTimeout int    // seconds after which tailing stops (0 = no timeout)
	timeSince   int64  // only show logs newer than this many seconds (0 = from start)
	stopString  string // stop tailing once this substring is seen in a log line
	onlyFailed  bool   // tail only pods that are not in the Running phase
	container   string // tail only this container (empty = all containers)

	debug bool

	out io.Writer
	env *cli.EnvSettings
	cfg *action.Configuration
}

var rtlHelp = `tail logs of a release in real time`

func NewRtLogsCmd(cfg *action.Configuration, out io.Writer, envs *cli.EnvSettings) *cobra.Command {
	rtl := &rtLogsCmd{
		out: out,
		env: envs,
		cfg: cfg,
	}

	cmd := &cobra.Command{
		Use:   "rtlogs [flags] RELEASE",
		Short: "rtlogs tails release logs in real time",
		Long:  rtlHelp,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rtl.release = args[0]
			return rtl.run(cmd.Context())
		},
	}

	f := cmd.Flags()
	f.IntVar(&rtl.stopTimeout, "stop-timeout", 0, "stop tailing after N seconds")
	f.StringVar(&rtl.stopString, "stop-string", "", "stop tailing once this substring appears in a log line")
	f.Int64VarP(&rtl.timeSince, "time-since", "s", 0, "show logs newer than N seconds")
	f.BoolVarP(&rtl.onlyFailed, "only-failed", "o", false, "tail only pods that are not Running")
	f.StringVarP(&rtl.container, "container", "c", "", "tail only this container (default: all)")
	f.BoolVarP(&rtl.debug, "debug", "d", false, "enable debug output")

	return cmd
}

// run resolves the release, builds a Kubernetes client and collects logs until the
// run is stopped by a timeout, a stop string, an interrupt signal, or all streams ending.
func (e *rtLogsCmd) run(parent context.Context) error {
	if parent == nil {
		parent = context.Background()
	}

	getRelease := action.NewGet(e.cfg)

	if kctx := os.Getenv("HELM_KUBECONTEXT"); kctx != "" {
		e.env.KubeContext = kctx
	}

	res, err := getRelease.Run(e.release)
	if err != nil {
		return err
	}

	clientset, err := kubeclient.NewKubeClient(e.env.KubeContext, e.env.KubeConfig)
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	opts := collector.RtLogsOpts{
		StopTimeout: e.stopTimeout,
		StopString:  e.stopString,
		TimeSince:   e.timeSince,
		OnlyFailed:  e.onlyFailed,
		Container:   e.container,
		Debug:       e.debug,
	}

	c := collector.Collector{
		KubeClient:     clientset,
		ReleaseInfo:    res,
		Opts:           &opts,
		Ctx:            ctx,
		CancelFunction: cancel,
		Out:            e.out,
	}

	if err := c.CollectLogs(); err != nil {
		return fmt.Errorf("collect logs: %w", err)
	}
	return nil
}
