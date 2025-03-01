package eventwriter

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/lager"
	"github.com/StefanPostma/dynatrace-firehose-nozzle/utils"
)

type SplunkConfig struct {
	Host    string
	Token   string
	Index   string
	Fields  map[string]string
	SkipSSL bool
	Debug   bool
	Version string

	Logger lager.Logger
}

type SplunkEvent struct {
	httpClient     *http.Client
	config         *SplunkConfig
	BodyBufferSize utils.Counter
	SentEventCount utils.Counter
}

func NewSplunkEvent(config *SplunkConfig) Writer {
	httpClient := cfhttp.NewClient()
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: config.SkipSSL, MinVersion: tls.VersionTLS12},
	}
	httpClient.Transport = tr

	return &SplunkEvent{
		httpClient:     httpClient,
		config:         config,
		BodyBufferSize: &utils.NopCounter{},
		SentEventCount: &utils.NopCounter{},
	}
}

func (s *SplunkEvent) Write(events []map[string]interface{}) (error, uint64) {
	bodyBuffer := new(bytes.Buffer)
	count := uint64(len(events))
	for i, event := range events {

		if _, ok := event["index"]; !ok {
			if event["event"].(map[string]interface{})["info_splunk_index"] != nil {
				event["index"] = event["event"].(map[string]interface{})["info_splunk_index"]
			} else if s.config.Index != "" {
				event["index"] = s.config.Index
			}
		}

		if len(s.config.Fields) > 0 {
			event["fields"] = s.config.Fields
		}

		eventJson, err := json.Marshal(event)
		if err == nil {
			bodyBuffer.Write(eventJson)
			if i < len(events)-1 {
				bodyBuffer.Write([]byte("\n\n"))
			}
		} else {
			s.config.Logger.Error("Error marshalling event", err,
				lager.Data{
					"event": fmt.Sprintf("%+v", event),
				},
			)
		}
	}

	if s.config.Debug {
		bodyString := bodyBuffer.String()
		return s.dump(bodyString), count
	} else {
		bodyBytes := bodyBuffer.Bytes()
		s.SentEventCount.Add(count)
		return s.send(&bodyBytes), count
	}
}

func (s *SplunkEvent) send(postBody *[]byte) error {
	endpoint := fmt.Sprintf("%s/api/v2/logs/ingest", s.config.Host)
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(*postBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json;charset=utf-8")
	//req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Authorization", fmt.Sprintf("Api-Token %s", s.config.Token))
	
	//Add app headers for HEC telemetry
	//req.Header.Set("log.source", "Dynatrace Firehose Nozzle")
	//req.Header.Set("__splunk_app_version", s.config.Version)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		responseBody, _ := io.ReadAll(resp.Body)
		return errors.New(fmt.Sprintf("Non-ok response code [%d] from Dynatrace: %s", resp.StatusCode, responseBody))
	} else {
		//Draining the response buffer, so that the same connection can be reused the next time
		_, err := io.Copy(io.Discard, resp.Body)
		if err != nil {
			s.config.Logger.Error("Error discarding response body", err)
		}
	}
	s.BodyBufferSize.Add(uint64(len(*postBody)))

	return nil
}

// To dump the event on stdout instead of Splunk, in case of 'debug' mode
func (s *SplunkEvent) dump(eventString string) error {
	fmt.Println(string(eventString))

	return nil
}
