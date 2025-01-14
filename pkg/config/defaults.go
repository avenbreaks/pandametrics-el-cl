package config

var (
	DefaultLogLevel       string = "info"
	DefaultInitSlot       int    = 0
	DefaultFinalSlot      int    = 0
	DefaultBnEndpoint     string = ""
	DefaultElEndpoint     string = ""
	DefaultDBUrl          string = "postgres://user:password@localhost:5432/goteth"
	DefaultDownloadMode   string = "finalized"
	DefaultWorkerNum      int    = 4
	DefaultDbWorkerNum    int    = 4
	DefaultMetrics        string = "epoch,block"
	DefaultPrometheusPort int    = 9080
)
