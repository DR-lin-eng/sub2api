package ffi

import "sync/atomic"

type PathMetricsSnapshot struct {
	Calls     int64 `json:"calls"`
	RustHits  int64 `json:"rust_hits"`
	Fallbacks int64 `json:"fallbacks"`
}

type MetricsSnapshot struct {
	Total          PathMetricsSnapshot `json:"total"`
	Hash           PathMetricsSnapshot `json:"hash"`
	EventParse     PathMetricsSnapshot `json:"event_parse"`
	EventPredicate PathMetricsSnapshot `json:"event_predicate"`
	PayloadMutate  PathMetricsSnapshot `json:"payload_mutate"`
	ToolCorrection PathMetricsSnapshot `json:"tool_correction"`
	Framing        PathMetricsSnapshot `json:"framing"`
	SSEBodySummary PathMetricsSnapshot `json:"sse_body_summary"`
}

type ffiPathMetrics struct {
	calls     atomic.Int64
	rustHits  atomic.Int64
	fallbacks atomic.Int64
}

type ffiRuntimeMetrics struct {
	total          ffiPathMetrics
	hash           ffiPathMetrics
	eventParse     ffiPathMetrics
	eventPredicate ffiPathMetrics
	payloadMutate  ffiPathMetrics
	toolCorrection ffiPathMetrics
	framing        ffiPathMetrics
	sseBodySummary ffiPathMetrics
}

type ffiMetricCategory int

const (
	ffiMetricHash ffiMetricCategory = iota
	ffiMetricEventParse
	ffiMetricEventPredicate
	ffiMetricPayloadMutate
	ffiMetricToolCorrection
	ffiMetricFraming
	ffiMetricSSEBodySummary
)

var defaultMetrics ffiRuntimeMetrics

func recordMetric(category ffiMetricCategory, rustHit bool) {
	path := defaultMetrics.path(category)
	path.calls.Add(1)
	defaultMetrics.total.calls.Add(1)
	if rustHit {
		path.rustHits.Add(1)
		defaultMetrics.total.rustHits.Add(1)
		return
	}
	path.fallbacks.Add(1)
	defaultMetrics.total.fallbacks.Add(1)
}

func SnapshotMetrics() MetricsSnapshot {
	return MetricsSnapshot{
		Total:          defaultMetrics.total.snapshot(),
		Hash:           defaultMetrics.hash.snapshot(),
		EventParse:     defaultMetrics.eventParse.snapshot(),
		EventPredicate: defaultMetrics.eventPredicate.snapshot(),
		PayloadMutate:  defaultMetrics.payloadMutate.snapshot(),
		ToolCorrection: defaultMetrics.toolCorrection.snapshot(),
		Framing:        defaultMetrics.framing.snapshot(),
		SSEBodySummary: defaultMetrics.sseBodySummary.snapshot(),
	}
}

func (m *ffiRuntimeMetrics) path(category ffiMetricCategory) *ffiPathMetrics {
	switch category {
	case ffiMetricHash:
		return &m.hash
	case ffiMetricEventParse:
		return &m.eventParse
	case ffiMetricEventPredicate:
		return &m.eventPredicate
	case ffiMetricPayloadMutate:
		return &m.payloadMutate
	case ffiMetricToolCorrection:
		return &m.toolCorrection
	case ffiMetricFraming:
		return &m.framing
	case ffiMetricSSEBodySummary:
		return &m.sseBodySummary
	default:
		return &m.total
	}
}

func (m *ffiPathMetrics) snapshot() PathMetricsSnapshot {
	return PathMetricsSnapshot{
		Calls:     m.calls.Load(),
		RustHits:  m.rustHits.Load(),
		Fallbacks: m.fallbacks.Load(),
	}
}

func resetMetricsForTest() {
	defaultMetrics.total.calls.Store(0)
	defaultMetrics.total.rustHits.Store(0)
	defaultMetrics.total.fallbacks.Store(0)
	defaultMetrics.hash.calls.Store(0)
	defaultMetrics.hash.rustHits.Store(0)
	defaultMetrics.hash.fallbacks.Store(0)
	defaultMetrics.eventParse.calls.Store(0)
	defaultMetrics.eventParse.rustHits.Store(0)
	defaultMetrics.eventParse.fallbacks.Store(0)
	defaultMetrics.eventPredicate.calls.Store(0)
	defaultMetrics.eventPredicate.rustHits.Store(0)
	defaultMetrics.eventPredicate.fallbacks.Store(0)
	defaultMetrics.payloadMutate.calls.Store(0)
	defaultMetrics.payloadMutate.rustHits.Store(0)
	defaultMetrics.payloadMutate.fallbacks.Store(0)
	defaultMetrics.toolCorrection.calls.Store(0)
	defaultMetrics.toolCorrection.rustHits.Store(0)
	defaultMetrics.toolCorrection.fallbacks.Store(0)
	defaultMetrics.framing.calls.Store(0)
	defaultMetrics.framing.rustHits.Store(0)
	defaultMetrics.framing.fallbacks.Store(0)
	defaultMetrics.sseBodySummary.calls.Store(0)
	defaultMetrics.sseBodySummary.rustHits.Store(0)
	defaultMetrics.sseBodySummary.fallbacks.Store(0)
}
