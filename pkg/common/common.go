package common

type KubeConfig struct {
	Context string
	File    string
}

type AppSettings struct {
	ReleaseName    string
	KubeConfigFile string
	KubeContext    string
	Namespace      string
	StopTimeout    int
	StopString     string
	TimeSince      int64
}
