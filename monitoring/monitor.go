package monitoring

import (
	"github.com/StefanPostma/dynatrace-firehose-nozzle/utils"
)

type MonitorFunc func() interface{}

type Monitor interface {
	RegisterFunc(string, MonitorFunc)
	RegisterCounter(string, utils.CounterType) utils.Counter
	Start()
	Stop() error
}

var (
	monitor Monitor = &NoMonitor{}
)

func RegisterFunc(id string, callerFunc MonitorFunc) {
	monitor.RegisterFunc(id, callerFunc)
}

func RegisterCounter(id string, varType utils.CounterType) utils.Counter {
	return monitor.RegisterCounter(id, varType)
}
