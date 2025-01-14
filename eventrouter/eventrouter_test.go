package eventrouter_test

import (
	. "github.com/StefanPostma/dynatrace-firehose-nozzle/eventrouter"
	"github.com/StefanPostma/dynatrace-firehose-nozzle/testing"
	"github.com/cloudfoundry/sonde-go/events"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("eventrouter", func() {

	var (
		r   Router
		err error

		origin        string
		deployment    string
		job           string
		jobIndex      string
		ip            string
		timestampNano int64
		msg           *events.Envelope
		eventType     events.Envelope_EventType

		memSink *testing.MemorySinkMock
		noCache *testing.MemoryCacheMock
	)

	BeforeEach(func() {
		noCache = testing.NewMemoryCacheMock()
		memSink = &testing.MemorySinkMock{}
		config := &Config{
			SelectedEvents: "LogMessage,HttpStart,HttpStop,HttpStartStop,ValueMetric,CounterEvent,Error,ContainerMetric",
		}
		r, err = New(noCache, memSink, config)
		Ω(err).ShouldNot(HaveOccurred())

		timestampNano = 1467040874046121775
		deployment = "cf-warden"
		jobIndex = "85c9ff80-e99b-470b-a194-b397a6e73913"
		ip = "10.244.0.22"
		appId := "f964a41c-76ac-42c1-b2ba-663da3ec22d5"
		sourcetype := "testing"
		mtype := events.LogMessage_OUT
		logMsg := &events.LogMessage{
			Message:        []byte("testing"),
			MessageType:    &mtype,
			Timestamp:      &timestampNano,
			AppId:          &appId,
			SourceType:     &sourcetype,
			SourceInstance: &sourcetype,
		}

		msg = &events.Envelope{
			Origin:     &origin,
			EventType:  &eventType,
			Timestamp:  &timestampNano,
			Deployment: &deployment,
			Job:        &job,
			Index:      &jobIndex,
			Ip:         &ip,
			LogMessage: logMsg,
		}
	})

	It("Route valid message", func() {
		eventTypes := []events.Envelope_EventType{
			events.Envelope_LogMessage, events.Envelope_HttpStart,
			events.Envelope_HttpStop, events.Envelope_HttpStartStop,
			events.Envelope_ValueMetric, events.Envelope_CounterEvent,
			events.Envelope_Error, events.Envelope_ContainerMetric,
		}
		for i, eType := range eventTypes {
			eventType = eType
			err := r.Route(msg)
			Ω(err).ShouldNot(HaveOccurred())
			Expect(len(memSink.Events)).To(Equal(i + 1))
		}
	})

	It("Route default selected message", func() {
		config := &Config{
			SelectedEvents: "",
		}
		r, err = New(noCache, memSink, config)
		Ω(err).ShouldNot(HaveOccurred())

		eventType = events.Envelope_LogMessage
		err := r.Route(msg)
		Ω(err).ShouldNot(HaveOccurred())
		Expect(len(memSink.Events)).To(Equal(1))
	})

	It("Invalid event", func() {
		config := &Config{
			SelectedEvents: "invalid-event",
		}
		_, err = New(noCache, memSink, config)
		Ω(err).Should(HaveOccurred())
	})
})
