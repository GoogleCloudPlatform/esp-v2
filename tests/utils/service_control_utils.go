// Copyright 2019 Google LLC
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
	"fmt"
	"math"
	"sort"
	"testing"

	"github.com/golang/protobuf/proto"

	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
	structpb "github.com/golang/protobuf/ptypes/struct"
	scpb "google.golang.org/genproto/googleapis/api/servicecontrol/v1"
	ltypepb "google.golang.org/genproto/googleapis/logging/type"
)

type ExpectedCheck struct {
	Version                string
	ServiceName            string
	ServiceConfigID        string
	ConsumerID             string
	OperationName          string
	CallerIp               string
	AndroidCertFingerprint string
	AndroidPackageName     string
	ApiKey                 string
	IosBundleID            string
	Referer                string
}

type ExpectedQuota struct {
	ConsumerID      string
	MethodName      string
	QuotaMetrics    map[string]int64
	QuotaMode       scpb.QuotaOperation_QuotaMode
	ServiceConfigID string
	ServiceName     string
}

type ExpectedReport struct {
	Aggregate         int64
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
	RequestMsgCounts  int64
	ResponseMsgCounts int64
	ResponseCode      int
	Referer           string
	StatusCode        string
	ErrorCause        string
	ErrorType         string
	FrontendProtocol  string
	BackendProtocol   string
	Platform          string
	JwtAuth           string
	RequestHeaders    string
	ResponseHeaders   string
	JwtPayloads       string
}

type distOptions struct {
	Buckets int32
	Growth  float64
	Scale   float64
}

type MetricCreator int

const (
	MTProducer MetricCreator = 1 + iota
	MTConsumer
	MTProducerByConsumer
	MTProducerUnderGrpcStream
	MTConsumerUnderGrpcStream
)

type MetricValueType int

const (
	Int64 MetricValueType = 1 + iota
	Distribution
)

type MetricValueInfo struct {
	MetricCreator   MetricCreator
	MetricValueType MetricValueType
	// Whether to use this metric when creating a ExpectedReport
	ShouldInit bool
}

var (
	timeDistOptions = distOptions{29, 2.0, 1e-6}
	sizeDistOptions = distOptions{8, 10.0, 1}
	randomMetrics   = map[string]MetricValueInfo{
		"serviceruntime.googleapis.com/api/consumer/total_latencies": {
			MetricCreator:   MTConsumer,
			MetricValueType: Distribution,
			ShouldInit:      true,
		},
		"serviceruntime.googleapis.com/api/producer/total_latencies": {
			MetricCreator:   MTProducer,
			MetricValueType: Distribution,
			ShouldInit:      true,
		},
		"serviceruntime.googleapis.com/api/producer/by_consumer/total_latencies": {
			MetricCreator:   MTProducerByConsumer,
			MetricValueType: Distribution,
			ShouldInit:      true,
		},
		"serviceruntime.googleapis.com/api/consumer/backend_latencies": {
			MetricCreator:   MTConsumer,
			MetricValueType: Distribution,
			ShouldInit:      true,
		},
		"serviceruntime.googleapis.com/api/producer/backend_latencies": {
			MetricCreator:   MTProducer,
			MetricValueType: Distribution,
			ShouldInit:      true,
		},
		"serviceruntime.googleapis.com/api/producer/by_consumer/backend_latencies": {
			MetricCreator:   MTProducerByConsumer,
			MetricValueType: Distribution,
			ShouldInit:      true,
		},
		"serviceruntime.googleapis.com/api/consumer/request_overhead_latencies": {
			MetricCreator:   MTConsumer,
			MetricValueType: Distribution,
			ShouldInit:      true,
		},
		"serviceruntime.googleapis.com/api/producer/request_overhead_latencies": {
			MetricCreator:   MTProducer,
			MetricValueType: Distribution,
			ShouldInit:      true,
		},
		"serviceruntime.googleapis.com/api/producer/by_consumer/request_overhead_latencies": {
			MetricCreator:   MTProducerByConsumer,
			MetricValueType: Distribution,
			ShouldInit:      true,
		},
		"serviceruntime.googleapis.com/api/consumer/streaming_durations": {
			MetricCreator:   MTConsumerUnderGrpcStream,
			MetricValueType: Distribution,
			ShouldInit:      true,
		},
		"serviceruntime.googleapis.com/api/producer/streaming_durations": {
			MetricCreator:   MTProducerUnderGrpcStream,
			MetricValueType: Distribution,
			ShouldInit:      true,
		},
		"serviceruntime.googleapis.com/api/consumer/request_sizes": {
			MetricCreator:   MTConsumer,
			MetricValueType: Distribution,
		},
		"serviceruntime.googleapis.com/api/consumer/response_sizes": {
			MetricCreator:   MTConsumer,
			MetricValueType: Distribution,
		},
		"serviceruntime.googleapis.com/api/consumer/request_bytes": {
			MetricCreator:   MTConsumer,
			MetricValueType: Int64,
		},
		"serviceruntime.googleapis.com/api/consumer/response_bytes": {
			MetricCreator:   MTConsumer,
			MetricValueType: Int64,
		},
		"serviceruntime.googleapis.com/api/producer/request_sizes": {
			MetricCreator:   MTProducer,
			MetricValueType: Distribution,
		},
		"serviceruntime.googleapis.com/api/producer/response_sizes": {
			MetricCreator:   MTProducer,
			MetricValueType: Distribution,
		},
		"serviceruntime.googleapis.com/api/producer/request_bytes": {
			MetricCreator:   MTProducer,
			MetricValueType: Int64,
		},
		"serviceruntime.googleapis.com/api/producer/response_bytes": {
			MetricCreator:   MTProducer,
			MetricValueType: Int64,
		},
		"serviceruntime.googleapis.com/api/producer/by_consumer/request_sizes": {
			MetricCreator:   MTProducerByConsumer,
			MetricValueType: Distribution,
		},
		"serviceruntime.googleapis.com/api/producer/by_consumer/response_sizes": {
			MetricCreator:   MTProducerByConsumer,
			MetricValueType: Distribution,
		},
	}
	randomLogEntries = []string{
		"timestamp",
		"request_latency_in_ms",
		"request_size_in_bytes",
		"response_size_in_bytes",
	}
	fakeDistVal  = 1000
	fakeInt64Val = 200
)

func CreateCheck(er *ExpectedCheck) scpb.CheckRequest {
	erPb := scpb.CheckRequest{
		ServiceName:     er.ServiceName,
		ServiceConfigId: er.ServiceConfigID,
		Operation: &scpb.Operation{
			OperationName: er.OperationName,
			ConsumerId:    er.ConsumerID,
			Labels: map[string]string{
				"servicecontrol.googleapis.com/user_agent":    "ESPv2",
				"servicecontrol.googleapis.com/service_agent": "ESPv2/" + er.Version,
			},
		},
	}
	if er.CallerIp != "" {
		erPb.Operation.Labels["servicecontrol.googleapis.com/caller_ip"] =
			er.CallerIp
	}

	if er.AndroidCertFingerprint != "" {
		erPb.Operation.Labels["servicecontrol.googleapis.com/android_cert_fingerprint"] = er.AndroidCertFingerprint
	}

	if er.AndroidPackageName != "" {
		erPb.Operation.Labels["servicecontrol.googleapis.com/android_package_name"] = er.AndroidPackageName
	}

	if er.IosBundleID != "" {
		erPb.Operation.Labels["servicecontrol.googleapis.com/ios_bundle_id"] = er.IosBundleID
	}

	if er.Referer != "" {
		erPb.Operation.Labels["servicecontrol.googleapis.com/referer"] = er.Referer
	}

	return erPb
}

func responseCodes(code int) (response, class string) {
	return fmt.Sprintf("%d", code),
		fmt.Sprintf("%dxx", code/100)
}

func createReportLabels(er *ExpectedReport) map[string]string {
	response, class := responseCodes(er.ResponseCode)
	labels := map[string]string{
		"servicecontrol.googleapis.com/service_agent": "ESPv2/" + er.Version,
		"servicecontrol.googleapis.com/user_agent":    "ESPv2",
		"serviceruntime.googleapis.com/api_method":    er.ApiMethod,
		"/response_code":       response,
		"/response_code_class": class,
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

func createLogEntry(er *ExpectedReport) *scpb.LogEntry {
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
	if er.RequestHeaders != "" {
		pl["request_headers"] = makeStringValue(er.RequestHeaders)
	}
	if er.ResponseHeaders != "" {
		pl["response_headers"] = makeStringValue(er.ResponseHeaders)
	}
	if er.JwtPayloads != "" {
		pl["jwt_payloads"] = makeStringValue(er.JwtPayloads)
	}
	pl["client_ip"] = makeStringValue("127.0.0.1")

	severity := ltypepb.LogSeverity_INFO
	if er.ResponseCode >= 400 {
		severity = ltypepb.LogSeverity_ERROR
	}

	return &scpb.LogEntry{
		Name:     "endpoints_log",
		Severity: severity,
		Payload: &scpb.LogEntry_StructPayload{
			&structpb.Struct{
				Fields: pl,
			},
		},
	}
}

func createInt64MetricSet(name string, value int64) *scpb.MetricValueSet {
	return &scpb.MetricValueSet{
		MetricName: name,
		MetricValues: []*scpb.MetricValue{
			&scpb.MetricValue{
				Value: &scpb.MetricValue_Int64Value{value},
			},
		},
	}
}

func createDistMetricSet(options *distOptions, name string, value int64) *scpb.MetricValueSet {
	buckets := make([]int64, options.Buckets+2)
	fValue := float64(value)
	idx := 0
	if fValue >= options.Scale {
		idx = 1 + int(math.Log(fValue/options.Scale)/math.Log(options.Growth))
		if idx >= len(buckets) {
			idx = len(buckets) - 1
		}
	}
	buckets[idx] = 1
	distValue := scpb.Distribution{
		Count:        1,
		BucketCounts: buckets,
		BucketOption: &scpb.Distribution_ExponentialBuckets_{
			&scpb.Distribution_ExponentialBuckets{
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
	return &scpb.MetricValueSet{
		MetricName: name,
		MetricValues: []*scpb.MetricValue{
			&scpb.MetricValue{
				Value: &scpb.MetricValue_DistributionValue{&distValue},
			},
		},
	}
}

// Update the metric with the value and aggregate it n times.
func updateDistMetricSet(m *scpb.MetricValueSet, fValue float64, n int64) {
	for _, v := range m.MetricValues {
		d := v.GetDistributionValue()
		o := d.GetExponentialBuckets()

		d.Mean = fValue
		d.Minimum = fValue
		d.Maximum = fValue
		d.Count = n
		d.SumOfSquaredDeviation = 0

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
		buckets[idx] = n
	}
}

type metricSetSorter []*scpb.MetricValueSet

func (a metricSetSorter) Len() int           { return len(a) }
func (a metricSetSorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a metricSetSorter) Less(i, j int) bool { return a[i].MetricName < a[j].MetricName }

func createOperation(er *ExpectedReport) *scpb.Operation {
	op := &scpb.Operation{
		OperationName: er.ApiMethod,
	}

	if er.ApiKey != "" {
		op.ConsumerId = "api_key:" + er.ApiKey
	}
	op.Labels = createReportLabels(er)
	return op
}

func createByConsumerOperation(er *ExpectedReport) *scpb.Operation {
	op := createOperation(er)

	// label serviceruntime.googleapis.com/consumer_project is only for by_consumer
	if er.ConsumerProjectID != "" {
		op.Labels["serviceruntime.googleapis.com/consumer_project"] = er.ConsumerProjectID
	}

	ms := []*scpb.MetricValueSet{
		createInt64MetricSet("serviceruntime.googleapis.com/api/producer/by_consumer/request_count", 1),
		createDistMetricSet(&sizeDistOptions,
			"serviceruntime.googleapis.com/api/producer/by_consumer/request_sizes", int64(fakeDistVal)),
		createDistMetricSet(&sizeDistOptions,
			"serviceruntime.googleapis.com/api/producer/by_consumer/response_sizes", int64(fakeDistVal)),
	}

	for name, t := range randomMetrics {
		if t.ShouldInit && t.MetricCreator == MTProducerByConsumer {
			ms = append(ms, createDistMetricSet(&timeDistOptions, name, int64(fakeDistVal)))
		}
	}

	if er.ErrorType != "" {
		ms = append(ms,
			createInt64MetricSet("serviceruntime.googleapis.com/api/producer/by_consumer/error_count", 1))
	}

	sort.Sort(metricSetSorter(ms))
	op.MetricValueSets = ms
	return op
}

// CreateReport makes a service_controller.proto ReportRequest out of an ExpectedReport
func CreateReport(er *ExpectedReport) scpb.ReportRequest {
	sendConsumer := er.ApiKey != ""
	sendByConsumer := er.ConsumerProjectID != ""

	op := createOperation(er)

	op.LogEntries = []*scpb.LogEntry{
		createLogEntry(er),
	}

	ms := []*scpb.MetricValueSet{
		createInt64MetricSet("serviceruntime.googleapis.com/api/producer/request_count", 1),
		createDistMetricSet(&sizeDistOptions,
			"serviceruntime.googleapis.com/api/producer/request_sizes", int64(fakeDistVal)),
		createDistMetricSet(&sizeDistOptions,
			"serviceruntime.googleapis.com/api/producer/response_sizes", int64(fakeDistVal)),
	}

	if er.RequestMsgCounts != 0 {
		ms = append(ms,
			createDistMetricSet(&sizeDistOptions, "serviceruntime.googleapis.com/api/producer/streaming_request_message_counts", er.RequestMsgCounts))
	}
	if er.ResponseMsgCounts != 0 {
		ms = append(ms,
			createDistMetricSet(&sizeDistOptions, "serviceruntime.googleapis.com/api/producer/streaming_response_message_counts", er.ResponseMsgCounts))
	}

	if sendConsumer {
		ms = append(ms,
			createInt64MetricSet("serviceruntime.googleapis.com/api/consumer/request_count", 1))
		ms = append(ms,
			createDistMetricSet(&sizeDistOptions,
				"serviceruntime.googleapis.com/api/consumer/request_sizes", int64(fakeDistVal)))
		ms = append(ms,
			createDistMetricSet(&sizeDistOptions,
				"serviceruntime.googleapis.com/api/consumer/response_sizes", int64(fakeDistVal)))

		if er.RequestMsgCounts != 0 {
			ms = append(ms,
				createDistMetricSet(&sizeDistOptions, "serviceruntime.googleapis.com/api/consumer/streaming_request_message_counts", er.RequestMsgCounts))
		}
		if er.ResponseMsgCounts != 0 {
			ms = append(ms,
				createDistMetricSet(&sizeDistOptions, "serviceruntime.googleapis.com/api/consumer/streaming_response_message_counts", er.ResponseMsgCounts))
		}
	}
	ms = append(ms,
		createInt64MetricSet("serviceruntime.googleapis.com/api/producer/request_bytes", int64(fakeInt64Val)))
	if sendConsumer {
		ms = append(ms,
			createInt64MetricSet("serviceruntime.googleapis.com/api/consumer/request_bytes", int64(fakeInt64Val)))
	}
	ms = append(ms,
		createInt64MetricSet("serviceruntime.googleapis.com/api/producer/response_bytes", int64(fakeInt64Val)))
	if sendConsumer {
		ms = append(ms,
			createInt64MetricSet("serviceruntime.googleapis.com/api/consumer/response_bytes", int64(fakeInt64Val)))
	}

	for name, t := range randomMetrics {
		if !t.ShouldInit {
			continue
		}

		if t.MetricCreator == MTProducer || sendConsumer && t.MetricCreator == MTConsumer {
			ms = append(ms, createDistMetricSet(&timeDistOptions, name, int64(fakeDistVal)))
		}
		if t.MetricCreator == MTProducerUnderGrpcStream || sendConsumer && t.MetricCreator == MTConsumerUnderGrpcStream {
			ms = append(ms, createDistMetricSet(&timeDistOptions, name, int64(fakeDistVal)))
		}
	}

	if er.ErrorType != "" {
		ms = append(ms,
			createInt64MetricSet("serviceruntime.googleapis.com/api/producer/error_count", 1))
		if sendConsumer {
			ms = append(ms,
				createInt64MetricSet("serviceruntime.googleapis.com/api/consumer/error_count", 1))
		}
	}

	sort.Sort(metricSetSorter(ms))
	op.MetricValueSets = ms

	erPb := scpb.ReportRequest{
		ServiceName:     er.ServiceName,
		ServiceConfigId: er.ServiceConfigID,
		Operations:      []*scpb.Operation{op},
	}
	if sendByConsumer {
		erPb.Operations = append(erPb.Operations, createByConsumerOperation(er))
	}
	return erPb
}

// Override the random metrics with a fixed value and aggregate it n times.
// Remove the random fields in LogEntry
func stripRandomFields(op *scpb.Operation, n int64) error {
	// Clear some fields
	op.OperationId = ""
	op.StartTime = nil
	op.EndTime = nil

	for i, m := range op.MetricValueSets {

		if info, found := randomMetrics[m.MetricName]; found {
			switch info.MetricValueType {
			case Int64:
				op.MetricValueSets[i] = createInt64MetricSet(m.MetricName, int64(fakeInt64Val)*n)
			case Distribution:
				updateDistMetricSet(m, float64(fakeDistVal), n)
			}
		}
	}
	sort.Sort(metricSetSorter(op.MetricValueSets))

	for _, l := range op.LogEntries {
		l.Timestamp = nil
		for _, s := range randomLogEntries {
			delete(l.GetStructPayload().Fields, s)
		}
	}

	return nil
}

// UnmarshalCheckRequest returns proto CheckRequest given data.
func UnmarshalCheckRequest(data []byte) (*scpb.CheckRequest, error) {
	rr := &scpb.CheckRequest{}
	err := proto.Unmarshal(data, rr)
	if err != nil {
		return nil, err
	}
	return rr, nil
}

// UnmarshalQuotaRequest returns proto AllocateQuotaRequest given data.
func UnmarshalQuotaRequest(data []byte) (*scpb.AllocateQuotaRequest, error) {
	rr := &scpb.AllocateQuotaRequest{}
	err := proto.Unmarshal(data, rr)
	if err != nil {
		return nil, err
	}
	return rr, nil
}

// VerifyCheck verify if the response body is the expected CheckRequest.
// If the verification fails, it returns an error.
func VerifyCheck(body []byte, ec *ExpectedCheck) error {
	got, err := UnmarshalCheckRequest(body)
	if err != nil {
		return err
	}
	if err := stripRandomFields(got.Operation, 1); err != nil {
		return err
	}

	want := CreateCheck(ec)

	if diff := ProtoDiff(&want, got); diff != "" {
		return fmt.Errorf("Diff (-want +got):\n%v", diff)
	}

	return nil
}

// UnmarshalReportRequest returns proto ReportRequest given data.
func UnmarshalReportRequest(data []byte) (*scpb.ReportRequest, error) {
	rr := &scpb.ReportRequest{}
	err := proto.Unmarshal(data, rr)
	if err != nil {
		return nil, err
	}
	return rr, nil
}

// VerifyReportRequestOperationLabel verifies whether a ReportRequest has the correct
// value for the label specified
func VerifyReportRequestOperationLabel(body []byte, label, value string) error {
	got, err := UnmarshalReportRequest(body)
	if err != nil {
		return err
	}

	if len(got.Operations) == 0 {
		return fmt.Errorf("Report is missing Operations")

	}

	for _, op := range got.Operations {
		if gotValue, ok := op.Labels[label]; ok {
			if gotValue != value {
				return fmt.Errorf("Mismatched value for label %v:\nWant %v\nGot %v",
					label, value, gotValue)
			}
			return nil
		}
	}

	return fmt.Errorf("No operations contained label %v", label)
}

// VerifyReport verify if the response body is the expected ReportRequest.
// If the verification fails, it returns an error.
func VerifyReport(body []byte, er *ExpectedReport) error {
	got, err := UnmarshalReportRequest(body)

	if err != nil {
		return err
	}

	var n int64
	if er.Aggregate > 1 {
		n = er.Aggregate
	} else {
		n = 1
	}
	for _, op := range got.Operations {
		if err := stripRandomFields(op, n); err != nil {
			return err
		}
	}

	want := CreateReport(er)

	if er.Aggregate > 1 {
		AggregateReport(&want, er.Aggregate)
	}

	if diff := ProtoDiff(&want, got); diff != "" {
		return fmt.Errorf("Diff (-want +got):\n%v", diff)
	}
	return nil
}

// VerifyQuota verify if the response body is the expected AllocateQuotaRequest.
// If the verification fails, it returns an error.
func VerifyQuota(body []byte, er *ExpectedQuota) error {
	got, err := UnmarshalQuotaRequest(body)
	if err != nil {
		return err
	}

	got.AllocateOperation.OperationId = ""

	quotaMetrics := []*scpb.MetricValueSet{}
	for key, val := range er.QuotaMetrics {
		quotaMetrics = append(quotaMetrics, &scpb.MetricValueSet{
			MetricName: key,
			MetricValues: []*scpb.MetricValue{
				{
					Value: &scpb.MetricValue_Int64Value{
						Int64Value: val,
					},
				},
			},
		})
	}
	sort.Slice(quotaMetrics, func(i, j int) bool {
		return quotaMetrics[i].MetricName < quotaMetrics[j].MetricName
	})
	sort.Slice(got.AllocateOperation.QuotaMetrics, func(i, j int) bool {
		return got.AllocateOperation.QuotaMetrics[i].MetricName < got.AllocateOperation.QuotaMetrics[j].MetricName
	})

	want := scpb.AllocateQuotaRequest{
		ServiceName: er.ServiceName,
		AllocateOperation: &scpb.QuotaOperation{
			MethodName:   er.MethodName,
			ConsumerId:   er.ConsumerID,
			QuotaMetrics: quotaMetrics,
			QuotaMode:    er.QuotaMode,
			Labels: map[string]string{
				"servicecontrol.googleapis.com/service_agent": fmt.Sprintf("ESPv2/%s", ESPv2Version()),
				"servicecontrol.googleapis.com/user_agent":    "ESPv2",
				"servicecontrol.googleapis.com/caller_ip":     "127.0.0.1",
			},
		},
		ServiceConfigId: er.ServiceConfigID,
	}
	if diff := ProtoDiff(&want, got); diff != "" {
		return fmt.Errorf("Diff (-want +got):\n%v", diff)
	}
	return nil
}

// AggregateReport aggregates N report body into one, as
// all metric values * N, and its LowEntries appended N times.
func AggregateReport(pb *scpb.ReportRequest, n int64) {
	for _, op := range pb.Operations {
		for _, m := range op.MetricValueSets {
			for _, v := range m.MetricValues {
				switch x := v.Value.(type) {
				case *scpb.MetricValue_Int64Value:
					x.Int64Value *= n
				case *scpb.MetricValue_DistributionValue:
					dist := x.DistributionValue
					dist.Count *= n
					bs := make([]int64, len(dist.BucketCounts))
					for i := 0; i < len(dist.BucketCounts); i++ {
						bs[i] = n * dist.BucketCounts[i]
					}
					dist.BucketCounts = bs
				default:
				}
			}
		}
		if op.LogEntries != nil {
			logs := []*scpb.LogEntry{}
			// Duplicate logEntry N times.
			for i := 0; i < int(n); i++ {
				logs = append(logs, op.LogEntries[0])
			}
			op.LogEntries = logs
		}
	}
}

func CheckScRequest(t *testing.T, scRequests []*comp.ServiceRequest, wantScRequests []interface{}, desc string) {
	t.Helper()

	for i, wantScRequest := range wantScRequests {
		scRequest := scRequests[i]
		reqBody := scRequest.ReqBody
		switch wantScRequest.(type) {
		case *ExpectedCheck:
			if scRequest.ReqType != comp.CHECK_REQUEST {
				t.Errorf("Test (%s): failed, service control request %v: should be Check", desc, i)
			}
			if err := VerifyCheck(reqBody, wantScRequest.(*ExpectedCheck)); err != nil {
				t.Errorf("Test (%s): failed,  %v", desc, err)
			}
		case *ExpectedQuota:
			if scRequest.ReqType != comp.QUOTA_REQUEST {
				t.Errorf("Test (%s): failed, service control request %v: should be Quota", desc, i)
			}
			if err := VerifyQuota(reqBody, wantScRequest.(*ExpectedQuota)); err != nil {
				t.Errorf("Test (%s): failed,  %v", desc, err)
			}
		case *ExpectedReport:
			if scRequest.ReqType != comp.REPORT_REQUEST {
				t.Errorf("Test (%s): failed, service control request %v: should be Report", desc, i)
			}
			if err := VerifyReport(reqBody, wantScRequest.(*ExpectedReport)); err != nil {
				t.Errorf("Test (%s): failed,  %v", desc, err)
			}
		default:
			t.Fatalf("Test (%s): failed, unknown service control response type", desc)
		}
	}
}

func CheckAPIKey(t *testing.T, scCheck *comp.ServiceRequest, wantApiKey string, desc string) {
	if scCheck.ReqType != comp.CHECK_REQUEST {
		t.Errorf("Test (%s): failed, the service control request should be Check", desc)
	}

	body := scCheck.ReqBody
	got, err := UnmarshalCheckRequest(body)
	if err != nil {
		t.Fatalf("Test (%s): failed, %v: ", desc, err)
	}

	if gotApiKey := got.Operation.ConsumerId[8:]; gotApiKey != wantApiKey {
		t.Errorf("Test (%s): failed, expected apiKey: %s, got %s", desc, wantApiKey, gotApiKey)
	}
}

func VerifyServiceControlResp(desc string, wantScRequests []interface{}, scRequests []*comp.ServiceRequest) error {
	for i, wantScRequest := range wantScRequests {
		reqBody := scRequests[i].ReqBody
		switch wantScRequest.(type) {
		case *ExpectedCheck:
			if scRequests[i].ReqType != comp.CHECK_REQUEST {
				return fmt.Errorf("Test Desc(%s): service control request %v: should be Check", desc, i)
			}
			if err := VerifyCheck(reqBody, wantScRequest.(*ExpectedCheck)); err != nil {
				return err
			}
		case *ExpectedReport:
			if scRequests[i].ReqType != comp.REPORT_REQUEST {
				return fmt.Errorf("Test Desc(%s): service control request %v: should be Report", desc, i)
			}
			if err := VerifyReport(reqBody, wantScRequest.(*ExpectedReport)); err != nil {
				return err
			}
		default:
			return fmt.Errorf("Test Desc(%s): unknown service control response type", desc)
		}
	}
	return nil
}
