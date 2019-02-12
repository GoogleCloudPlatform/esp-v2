// Copyright 2018 Google Cloud Platform Proxy Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"sort"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/struct"
	sc "github.com/google/go-genproto/googleapis/api/servicecontrol/v1"
	"google.golang.org/genproto/googleapis/logging/type"
)

type ExpectedCheck struct {
	Version         string
	ServiceName     string
	ServiceConfigID string
	ConsumerID      string
	OperationName   string
	CallerIp        string
}

type ExpectedReport struct {
	Version           string
	ServiceName       string
	ServiceConfigID   string
	ApiVersion        string
	ApiMethod         string
	ApiKey            string
	ProducerProjectID string
	ConsumerProjectID string
	URL               string
	Location          string
	HttpMethod        string
	LogMessage        string
	RequestSize       int64
	ResponseSize      int64
	RequestBytes      int64
	ResponseBytes     int64
	ResponseCode      int
	Referer           string
	StatusCode        string
	ErrorCause        string
	ErrorType         string
	FrontendProtocol  string
	BackendProtocol   string
	Platform          string
	JwtAuth           string
}

type distOptions struct {
	Buckets int32
	Growth  float64
	Scale   float64
}

var (
	timeDistOptions = distOptions{29, 2.0, 1e-6}
	sizeDistOptions = distOptions{8, 10.0, 1}
	randomMatrics   = map[string]bool{
		"serviceruntime.googleapis.com/api/consumer/total_latencies":            true,
		"serviceruntime.googleapis.com/api/producer/total_latencies":            true,
		"serviceruntime.googleapis.com/api/consumer/backend_latencies":          true,
		"serviceruntime.googleapis.com/api/producer/backend_latencies":          true,
		"serviceruntime.googleapis.com/api/consumer/request_overhead_latencies": true,
		"serviceruntime.googleapis.com/api/producer/request_overhead_latencies": true,
		"serviceruntime.googleapis.com/api/consumer/streaming_durations":        true,
		"serviceruntime.googleapis.com/api/producer/streaming_durations":        true,
	}
	randomLogEntries = []string{
		"timestamp",
		"request_latency_in_ms",
	}
	fakeLatency = 1000
)

func CreateCheck(er *ExpectedCheck) sc.CheckRequest {
	erPb := sc.CheckRequest{
		ServiceName:     er.ServiceName,
		ServiceConfigId: er.ServiceConfigID,
		Operation: &sc.Operation{
			OperationName: er.OperationName,
			ConsumerId:    er.ConsumerID,
			Labels: map[string]string{
				"servicecontrol.googleapis.com/user_agent":    "ESP",
				"servicecontrol.googleapis.com/service_agent": "ESP/" + er.Version,
			},
		},
	}
	if er.CallerIp != "" {
		erPb.Operation.Labels["servicecontrol.googleapis.com/caller_ip"] =
			er.CallerIp
	}
	return erPb
}

func responseCodes(code int) (response, status, class string) {
	return fmt.Sprintf("%d", code),
		fmt.Sprintf("%d", HttpResponseCodeToStatusCode(code)),
		fmt.Sprintf("%dxx", code/100)
}

func createReportLabels(er *ExpectedReport) map[string]string {
	response, status, class := responseCodes(er.ResponseCode)
	labels := map[string]string{
		"servicecontrol.googleapis.com/service_agent": "ESP/" + er.Version,
		"servicecontrol.googleapis.com/user_agent":    "ESP",
		"serviceruntime.googleapis.com/api_method":    er.ApiMethod,
		"/response_code":                              response,
		"/status_code":                                status,
		"/response_code_class":                        class,
	}
	if er.StatusCode != "" {
		labels["/status_code"] = er.StatusCode
	}
	if er.ErrorType != "" {
		labels["/error_type"] = er.ErrorType
	}

	if er.Location != "" {
		labels["cloud.googleapis.com/location"] = er.Location
	}

	if er.FrontendProtocol != "" {
		labels["/protocol"] = er.FrontendProtocol
	} else {
		labels["/protocol"] = "unknown"
	}
	if er.BackendProtocol != "" {
		labels["servicecontrol.googleapis.com/backend_protocol"] = er.BackendProtocol
	}

	if er.ApiVersion != "" {
		labels["serviceruntime.googleapis.com/api_version"] = er.ApiVersion
	}

	if er.Platform != "" {
		labels["servicecontrol.googleapis.com/platform"] = er.Platform
	} else {
		labels["servicecontrol.googleapis.com/platform"] = "unknown"
	}

	if er.ApiKey != "" {
		labels["/credential_id"] = "apikey:" + er.ApiKey
	} else if er.JwtAuth != "" {
		labels["/credential_id"] = "jwtauth:" + er.JwtAuth
	}

	return labels
}

func makeStringValue(v string) *structpb.Value {
	return &structpb.Value{Kind: &structpb.Value_StringValue{v}}
}

func makeNumberValue(v int64) *structpb.Value {
	return &structpb.Value{Kind: &structpb.Value_NumberValue{float64(v)}}
}

func createLogEntry(er *ExpectedReport) *sc.LogEntry {
	pl := make(map[string]*structpb.Value)

	pl["api_method"] = makeStringValue(er.ApiMethod)
	pl["http_response_code"] = makeNumberValue(int64(er.ResponseCode))

	if er.ApiVersion != "" {
		pl["api_version"] = makeStringValue(er.ApiVersion)
	}
	if er.ProducerProjectID != "" {
		pl["producer_project_id"] = makeStringValue(er.ProducerProjectID)
	}
	if er.ApiKey != "" {
		pl["api_key"] = makeStringValue(er.ApiKey)
	}
	if er.Referer != "" {
		pl["referer"] = makeStringValue(er.Referer)
	}
	if er.Location != "" {
		pl["location"] = makeStringValue(er.Location)
	}
	if er.RequestSize != 0 {
		pl["request_size_in_bytes"] = makeNumberValue(er.RequestSize)
	}
	if er.ResponseSize != 0 {
		pl["response_size_in_bytes"] = makeNumberValue(er.ResponseSize)
	}
	if er.LogMessage != "" {
		pl["log_message"] = makeStringValue(er.LogMessage)
	}
	if er.URL != "" {
		pl["url"] = makeStringValue(er.URL)
	}
	if er.HttpMethod != "" {
		pl["http_method"] = makeStringValue(er.HttpMethod)
	}
	if er.ErrorCause != "" {
		pl["error_cause"] = makeStringValue(er.ErrorCause)
	}

	severity := ltype.LogSeverity_INFO
	if er.ResponseCode >= 400 {
		severity = ltype.LogSeverity_ERROR
	}

	return &sc.LogEntry{
		Name:     "endpoints_log",
		Severity: severity,
		Payload: &sc.LogEntry_StructPayload{
			&structpb.Struct{
				Fields: pl,
			},
		},
	}
}

func createInt64MetricSet(name string, value int64) *sc.MetricValueSet {
	return &sc.MetricValueSet{
		MetricName: name,
		MetricValues: []*sc.MetricValue{
			&sc.MetricValue{
				Value: &sc.MetricValue_Int64Value{value},
			},
		},
	}
}

func createDistMetricSet(options *distOptions, name string, value int64) *sc.MetricValueSet {
	buckets := make([]int64, options.Buckets+2, options.Buckets+2)
	fValue := float64(value)
	idx := 0
	if fValue >= options.Scale {
		idx = 1 + int(math.Log(fValue/options.Scale)/math.Log(options.Growth))
		if idx >= len(buckets) {
			idx = len(buckets) - 1
		}
	}
	buckets[idx] = 1
	distValue := sc.Distribution{
		Count:        1,
		BucketCounts: buckets,
		BucketOption: &sc.Distribution_ExponentialBuckets_{
			&sc.Distribution_ExponentialBuckets{
				NumFiniteBuckets: options.Buckets,
				GrowthFactor:     options.Growth,
				Scale:            options.Scale,
			},
		},
	}

	if value != 0 {
		distValue.Mean = fValue
		distValue.Minimum = fValue
		distValue.Maximum = fValue
	}
	return &sc.MetricValueSet{
		MetricName: name,
		MetricValues: []*sc.MetricValue{
			&sc.MetricValue{
				Value: &sc.MetricValue_DistributionValue{&distValue},
			},
		},
	}
}

func updateDistMetricSet(m *sc.MetricValueSet, fValue float64) {
	for _, v := range m.MetricValues {
		d := v.GetDistributionValue()
		o := d.GetExponentialBuckets()

		d.Mean = fValue
		d.Minimum = fValue
		d.Maximum = fValue

		buckets := d.BucketCounts
		idx := 0
		if fValue >= o.Scale {
			idx = 1 + int(math.Log(fValue/o.Scale)/math.Log(o.GrowthFactor))
			if idx >= len(buckets) {
				idx = len(buckets) - 1
			}
		}
		for i, _ := range buckets {
			buckets[i] = 0
		}
		buckets[idx] = 1
	}
}

type metricSetSorter []*sc.MetricValueSet

func (a metricSetSorter) Len() int           { return len(a) }
func (a metricSetSorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a metricSetSorter) Less(i, j int) bool { return a[i].MetricName < a[j].MetricName }

func createOperation(er *ExpectedReport) *sc.Operation {
	op := &sc.Operation{
		OperationName: er.ApiMethod,
	}

	if er.ApiKey != "" {
		op.ConsumerId = "api_key:" + er.ApiKey
	}
	op.Labels = createReportLabels(er)
	return op
}

func createByConsumerOperation(er *ExpectedReport) *sc.Operation {
	op := createOperation(er)

	// label serviceruntime.googleapis.com/consumer_project is only for by_consumer
	if er.ConsumerProjectID != "" {
		op.Labels["serviceruntime.googleapis.com/consumer_project"] = er.ConsumerProjectID
	}

	ms := []*sc.MetricValueSet{
		createInt64MetricSet("serviceruntime.googleapis.com/api/producer/by_consumer/request_count", 1),
		createDistMetricSet(&sizeDistOptions,
			"serviceruntime.googleapis.com/api/producer/by_consumer/request_sizes", er.RequestSize),
	}

	// TODO(qiwzhang): add latency metrics b/123950502
	//	for name, _ := range byConsumerRandomMatrics {
	//		ms = append(ms, createDistMetricSet(&timeDistOptions, name, int64(fakeLatency)))
	//	}

	if er.ResponseSize != 0 {
		ms = append(ms,
			createDistMetricSet(&sizeDistOptions,
				"serviceruntime.googleapis.com/api/producer/by_consumer/response_sizes", er.ResponseSize))
	}
	if er.ErrorType != "" {
		ms = append(ms,
			createInt64MetricSet("serviceruntime.googleapis.com/api/producer/by_consumer/error_count", 1))
	}

	sort.Sort(metricSetSorter(ms))
	op.MetricValueSets = ms
	return op
}

func CreateReport(er *ExpectedReport) sc.ReportRequest {
	send_consumer := er.ApiKey != ""

	op := createOperation(er)

	op.LogEntries = []*sc.LogEntry{
		createLogEntry(er),
	}

	ms := []*sc.MetricValueSet{}

	ms = append(ms,
		createInt64MetricSet("serviceruntime.googleapis.com/api/producer/request_count", 1))
	if send_consumer {
		ms = append(ms,
			createInt64MetricSet("serviceruntime.googleapis.com/api/consumer/request_count", 1))
	}

	ms = append(ms,
		createDistMetricSet(&sizeDistOptions,
			"serviceruntime.googleapis.com/api/producer/request_sizes", er.RequestSize))
	if send_consumer {
		ms = append(ms,
			createDistMetricSet(&sizeDistOptions,
				"serviceruntime.googleapis.com/api/consumer/request_sizes", er.RequestSize))
	}

	ms = append(ms,
		createInt64MetricSet("serviceruntime.googleapis.com/api/producer/request_bytes", er.RequestBytes))
	if send_consumer {
		ms = append(ms,
			createInt64MetricSet("serviceruntime.googleapis.com/api/consumer/request_bytes", er.RequestBytes))
	}

	// TODO(qiwzhang): add latency metrics b/123950502
	//	for name, _ := range randomMatrics {
	//		ms = append(ms, createDistMetricSet(&timeDistOptions, name, int64(fakeLatency)))
	//	}

	if er.ResponseSize != 0 {
		ms = append(ms,
			createDistMetricSet(&sizeDistOptions,
				"serviceruntime.googleapis.com/api/producer/response_sizes", er.ResponseSize))
		if send_consumer {
			ms = append(ms,
				createDistMetricSet(&sizeDistOptions,
					"serviceruntime.googleapis.com/api/consumer/response_sizes", er.ResponseSize))
		}

		ms = append(ms,
			createInt64MetricSet("serviceruntime.googleapis.com/api/producer/response_bytes", er.ResponseBytes))
		if send_consumer {
			ms = append(ms,
				createInt64MetricSet("serviceruntime.googleapis.com/api/consumer/response_bytes", er.ResponseBytes))
		}
	}
	if er.ErrorType != "" {
		ms = append(ms,
			createInt64MetricSet("serviceruntime.googleapis.com/api/producer/error_count", 1))
		if send_consumer {
			ms = append(ms,
				createInt64MetricSet("serviceruntime.googleapis.com/api/consumer/error_count", 1))
		}
	}
	sort.Sort(metricSetSorter(ms))
	op.MetricValueSets = ms

	erPb := sc.ReportRequest{
		ServiceName:     er.ServiceName,
		ServiceConfigId: er.ServiceConfigID,
		Operations:      []*sc.Operation{op},
	}
	if send_consumer {
		erPb.Operations = append(erPb.Operations, createByConsumerOperation(er))
	}
	return erPb
}

func stripRandomFields(op *sc.Operation) {
	// Clear some fields
	op.OperationId = ""
	op.StartTime = nil
	op.EndTime = nil

	for _, m := range op.MetricValueSets {
		if _, found := randomMatrics[m.MetricName]; found {
			updateDistMetricSet(m, float64(fakeLatency))
		}
	}
	sort.Sort(metricSetSorter(op.MetricValueSets))

	for _, l := range op.LogEntries {
		l.Timestamp = nil
		for _, s := range randomLogEntries {
			delete(l.GetStructPayload().Fields, s)
		}
	}
}

func compareProto(t, e proto.Message) bool {
	if proto.Equal(t, e) {
		return true
	}
	var ts bytes.Buffer
	if err := proto.MarshalText(&ts, t); err == nil {
		fmt.Println("=== Got:\n", ts.String())
	}
	var es bytes.Buffer
	if err := proto.MarshalText(&es, e); err == nil {
		fmt.Println("=== Expected:\n", es.String())
	}
	return false
}

func VerifyCheck(body []byte, er *ExpectedCheck) bool {
	cr := sc.CheckRequest{}
	err := proto.Unmarshal(body, &cr)
	if err != nil {
		log.Println("failed to parse body into CheckRequest.")
		return false
	}
	stripRandomFields(cr.Operation)

	erPb := CreateCheck(er)
	return compareProto(&cr, &erPb)
}

func VerifyReport(body []byte, er *ExpectedReport) bool {
	cr := sc.ReportRequest{}
	err := proto.Unmarshal(body, &cr)
	if err != nil {
		log.Println("failed to parse body into ReportRequest.")
		return false
	}
	for _, op := range cr.Operations {
		stripRandomFields(op)
	}

	erPb := CreateReport(er)
	return compareProto(&cr, &erPb)
}
