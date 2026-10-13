package metrics_test

import (
	"io"
	"reflect"
	"syscall"
	"testing"

	metrics "github.com/hashicorp/go-metrics"
	compat "github.com/hashicorp/go-metrics/compat"
)

var (
	_ func(*metrics.InmemSink, syscall.Signal, io.Writer) *metrics.InmemSignal = metrics.NewInmemSignal
	_ func(*compat.InmemSink, syscall.Signal, io.Writer) *compat.InmemSignal   = compat.NewInmemSignal
)

func TestSignalServicesRefuse(t *testing.T) {
	for name, create := range map[string]func(){
		"root":           func() { metrics.NewInmemSignal(nil, metrics.DefaultSignal, io.Discard) },
		"root-default":   func() { metrics.DefaultInmemSignal(nil) },
		"compat":         func() { compat.NewInmemSignal(nil, compat.DefaultSignal, io.Discard) },
		"compat-default": func() { compat.DefaultInmemSignal(nil) },
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if got := recover(); got != "gomad: in-memory metrics signal service is unsupported" {
					t.Fatalf("signal service refusal = %v", got)
				}
			}()
			create()
		})
	}
}

type metric struct {
	Kind   string
	Key    []string
	Value  float32
	Labels []metrics.Label
}

type recordingSink struct {
	metrics.BlackholeSink
	metrics []metric
}

func (sink *recordingSink) SetGaugeWithLabels(key []string, value float32, labels []metrics.Label) {
	sink.metrics = append(sink.metrics, metric{"gauge", key, value, labels})
}

func (sink *recordingSink) IncrCounterWithLabels(key []string, value float32, labels []metrics.Label) {
	sink.metrics = append(sink.metrics, metric{"counter", key, value, labels})
}

func (sink *recordingSink) AddSampleWithLabels(key []string, value float32, labels []metrics.Label) {
	sink.metrics = append(sink.metrics, metric{"sample", key, value, labels})
}

func TestOrdinaryMetricsEmissionSurvives(t *testing.T) {
	sink := &recordingSink{}
	config := &metrics.Config{EnableRuntimeMetrics: false, FilterDefault: true}
	instance, err := metrics.NewGlobal(config, sink)
	if err != nil {
		t.Fatal(err)
	}
	labels := []metrics.Label{{Name: "zone", Value: "local"}}
	instance.SetGaugeWithLabels([]string{"gauge"}, 3, labels)
	compat.IncrCounterWithLabels([]string{"counter"}, 2, labels)
	compat.AddSampleWithLabels([]string{"sample"}, 5, labels)
	want := []metric{
		{"gauge", []string{"gauge"}, 3, labels},
		{"counter", []string{"counter"}, 2, labels},
		{"sample", []string{"sample"}, 5, labels},
	}
	if !reflect.DeepEqual(sink.metrics, want) {
		t.Fatalf("ordinary metrics = %#v, want %#v", sink.metrics, want)
	}
}
