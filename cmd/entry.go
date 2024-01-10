package cmd

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"os"

	"helm-rt-logs/pkg/collector"
	"helm-rt-logs/pkg/kubeclient"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type rtLogsCmd struct {
	release                 string // release name
	stopTimeout             int    // timeout to stop the tail
	timeSince               int64  // time since to start the tail
	stopString              string // string to stop the tail
	waitingFailedPodTimeout int    // waiting for Running phase in seconds timeout
	onlyFailed              bool   // tail only non-running pods

	debug bool // for debug, you know

	out io.Writer // output stream
	env *cli.EnvSettings
	cfg *action.Configuration // action configuration

}

var (
	rtlHelp = `
		tail logs of a release
	`
)

func NewRtLogsCmd(cfg *action.Configuration, out io.Writer, envs *cli.EnvSettings) *cobra.Command {

	rtl := &rtLogsCmd{
		out: out,
		env: envs,
	}

	cmd := &cobra.Command{
		Use:   "rtlogs [flags] RELEASE",
		Short: "rtlogs tail logs real time",
		Long:  rtlHelp,
		RunE: func(cmd *cobra.Command, args []string) error {
			rtl.cfg = cfg
			if len(args) != 1 {
				return fmt.Errorf("This command neeeds 1 argument: release name")
			}
			rtl.release = args[0]
			return rtl.run()
		},
	}
	f := cmd.Flags()

	f.IntVar(&rtl.stopTimeout, "stop-timeout", 0, "timeout to stop the logs, in Seconds!")
	f.StringVar(&rtl.stopString, "stop-string", "", "string to stop the logs")
	f.Int64VarP(&rtl.timeSince, "time-since", "s", 0, "time since to start the logs")
	f.IntVarP(&rtl.waitingFailedPodTimeout, "wait-fail-pods-timeout", "t", 60, "waiting for Running phase pods timeout")
	f.BoolVarP(&rtl.onlyFailed, "only-failed", "o", false, "tail only pods that have non Running phase")
	f.BoolVarP(&rtl.debug, "debug", "d", false, "enable debug")

	return cmd
}

func buildConfigFromFlags(context, kubeconfigPath string) (*rest.Config, error) {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{
			CurrentContext: context,
		}).ClientConfig()
}

func (e *rtLogsCmd) run() error {

	getRelease := action.NewGet(e.cfg)

	if ctx := os.Getenv("HELM_KUBECONTEXT"); ctx != "" {
		e.env.KubeContext = ctx
	}

	res, err := getRelease.Run(e.release)
	if err != nil {
		return err
	}

	clientset, err := kubeclient.NewKubeClient(e.env.KubeContext, e.env.KubeConfig)
	if err != nil {
		return err
	}

	c := collector.RtLogsOpts{
		StopTimeout:             e.stopTimeout,
		StopString:              e.stopString,
		TimeSince:               e.timeSince,
		WaitingFailedPodTimeout: e.waitingFailedPodTimeout,
		OnlyFailed:              e.onlyFailed,
		Debug:                   e.debug,
	}

	ctx, cancel := context.WithCancel(context.Background())

	newCollector := collector.Collector{
		KubeClient:     clientset,
		ReleaseInfo:    res,
		Opts:           &c,
		Ctx:            ctx,
		CancelFunction: cancel,
	}

	err = newCollector.CollectLogs()

	if err != nil {
		return err
	}

	return nil
}
