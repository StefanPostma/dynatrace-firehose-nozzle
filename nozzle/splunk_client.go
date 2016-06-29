package nozzle

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/cloudfoundry-incubator/cf_http"
	"github.com/pivotal-golang/lager"
)

type SplunkEvent struct {
	Time       string `json:"time,omitempty"`
	Host       string `json:"host,omitempty"`
	Source     string `json:"source,omitempty"`
	SourceType string `json:"sourcetype,omitempty"`
	Index      string `json:"index,omitempty"`

	Event interface{} `json:"event"`
}

type SplunkClient interface {
	PostSingle(*SplunkEvent) error
	PostBatch([]*SplunkEvent) error
}

type splunkClient struct {
	httpClient  *http.Client
	splunkToken string
	splunkHost  string
	logger      lager.Logger
}

func NewSplunkClient(splunkToken string, splunkHost string, insecureSkipVerify bool, logger lager.Logger) SplunkClient {
	httpClient := cf_http.NewClient()
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSkipVerify},
	}
	httpClient.Transport = tr

	return &splunkClient{
		httpClient:  httpClient,
		splunkToken: splunkToken,
		splunkHost:  splunkHost,
		logger:      logger,
	}
}

func (s *splunkClient) PostSingle(event *SplunkEvent) error {
	postBody, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return s.send(&postBody)
}

func (s *splunkClient) PostBatch(events []*SplunkEvent) error {
	bodyBuffer := new(bytes.Buffer)
	for i, event := range events {
		eventJson, err := json.Marshal(event)
		if err == nil {
			bodyBuffer.Write(eventJson)
			if i < len(events)-1 {
				bodyBuffer.Write([]byte("\n\n"))
			}
		} else {
			s.logger.Error("Error marshalling event", err,
				lager.Data{
					"event": fmt.Sprintf("%+v", event),
				},
			)
		}
	}

	bodyBytes := bodyBuffer.Bytes()
	return s.send(&bodyBytes)
}

func (s *splunkClient) send(postBody *[]byte) error {
	endpoint := fmt.Sprintf("%s/services/collector", s.splunkHost)
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(*postBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Splunk %s", s.splunkToken))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		responseBody, _ := ioutil.ReadAll(resp.Body)
		return errors.New(fmt.Sprintf("Non-ok response code [%d] from splunk: %s", resp.StatusCode, responseBody))
	}

	return nil
}