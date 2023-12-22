package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"helm-rt-logs/pkg/collector"
	"helm-rt-logs/pkg/kubeclient"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"io"
)

type rtLogsCmd struct {
	release     string    // release name
	out         io.Writer // output stream
	stopTimeout int       // timeout to stop the logs
	timeSince   int64     // time since to start the logs
	stopString  string    // string to stop the logs
	env         *cli.EnvSettings
	cfg         *action.Configuration // action configuration
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

	return cmd
}

func (e *rtLogsCmd) run() error {

	getRelease := action.NewGet(e.cfg)
	// getRelease.Version = 0
	res, err := getRelease.Run(e.release)
	if err != nil {
		return err
	}

	clientset, err := kubeclient.NewKubeClient(e.env.KubeContext, e.env.KubeConfig)
	if err != nil {
		return err
	}

	c := collector.RtLogsOpts{
		StopTimeout: e.stopTimeout,
		StopString:  e.stopString,
		TimeSince:   e.timeSince,
	}

	collector.CollectLogs(collector.Collector{
		KubeClient:  clientset,
		ReleaseInfo: res,
		Opts:        &c,
	})

	return nil
}
