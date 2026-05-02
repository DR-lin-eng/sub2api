package runtimeobs

import "sync/atomic"

// GnetHTTP1Snapshot is an in-process snapshot for ingress/runtime diagnostics.
// Values are cumulative since process start and intentionally cheap to update
// from the gnet hot path.
type GnetHTTP1Snapshot struct {
	HTTP1ClassifiedTotal     int64   `json:"http1_classified_total"`
	H2CClassifiedTotal       int64   `json:"h2c_classified_total"`
	SidecarClassifiedTotal   int64   `json:"sidecar_classified_total"`
	ClassifyErrorTotal       int64   `json:"classify_error_total"`
	EnqueueDropTotal         int64   `json:"enqueue_drop_total"`
	HTTP1EnqueueDropTotal    int64   `json:"http1_enqueue_drop_total"`
	H2CEnqueueDropTotal      int64   `json:"h2c_enqueue_drop_total"`
	SidecarEnqueueDropTotal  int64   `json:"sidecar_enqueue_drop_total"`
	ResponseTotal            int64   `json:"response_total"`
	BufferedResponseTotal    int64   `json:"buffered_response_total"`
	InlineBufferHitTotal     int64   `json:"inline_buffer_hit_total"`
	HeapBufferSpillTotal     int64   `json:"heap_buffer_spill_total"`
	ContentLengthAutoTotal   int64   `json:"content_length_auto_total"`
	ChunkedFallbackTotal     int64   `json:"chunked_fallback_total"`
	ChunkedFlushTotal        int64   `json:"chunked_flush_total"`
	ChunkedHeaderTotal       int64   `json:"chunked_header_total"`
	DirectWriteAfterFlush    int64   `json:"direct_write_after_flush_total"`
	AsyncWriteTotal          int64   `json:"async_write_total"`
	BufferedResponseRatio    float64 `json:"buffered_response_ratio"`
	InlineBufferHitRatio     float64 `json:"inline_buffer_hit_ratio"`
	ChunkedFallbackRatio     float64 `json:"chunked_fallback_ratio"`
	HTTP1ClassificationRatio float64 `json:"http1_classification_ratio"`
}

type gnetHTTP1Metrics struct {
	http1Classified    atomic.Int64
	h2cClassified      atomic.Int64
	sidecarClassified  atomic.Int64
	classifyError      atomic.Int64
	enqueueDrop        atomic.Int64
	http1EnqueueDrop   atomic.Int64
	h2cEnqueueDrop     atomic.Int64
	sidecarEnqueueDrop atomic.Int64

	responseTotal         atomic.Int64
	bufferedResponse      atomic.Int64
	inlineBufferHit       atomic.Int64
	heapBufferSpill       atomic.Int64
	contentLengthAuto     atomic.Int64
	chunkedFallback       atomic.Int64
	chunkedFlush          atomic.Int64
	chunkedHeader         atomic.Int64
	directWriteAfterFlush atomic.Int64
	asyncWrite            atomic.Int64
}

var gnetHTTP1 gnetHTTP1Metrics

func RecordGnetHTTP1Classified() {
	gnetHTTP1.http1Classified.Add(1)
}

func RecordGnetH2CClassified() {
	gnetHTTP1.h2cClassified.Add(1)
}

func RecordGnetSidecarClassified() {
	gnetHTTP1.sidecarClassified.Add(1)
}

func RecordGnetClassifyError() {
	gnetHTTP1.classifyError.Add(1)
}

func RecordGnetHTTP1EnqueueDrop() {
	gnetHTTP1.enqueueDrop.Add(1)
	gnetHTTP1.http1EnqueueDrop.Add(1)
}

func RecordGnetH2CEnqueueDrop() {
	gnetHTTP1.enqueueDrop.Add(1)
	gnetHTTP1.h2cEnqueueDrop.Add(1)
}

func RecordGnetSidecarEnqueueDrop() {
	gnetHTTP1.enqueueDrop.Add(1)
	gnetHTTP1.sidecarEnqueueDrop.Add(1)
}

func RecordGnetHTTP1Response() {
	gnetHTTP1.responseTotal.Add(1)
}

func RecordGnetHTTP1BufferedResponse(inlineBody bool, autoContentLength bool) {
	gnetHTTP1.bufferedResponse.Add(1)
	if inlineBody {
		gnetHTTP1.inlineBufferHit.Add(1)
	}
	if autoContentLength {
		gnetHTTP1.contentLengthAuto.Add(1)
	}
}

func RecordGnetHTTP1HeapBufferSpill() {
	gnetHTTP1.heapBufferSpill.Add(1)
}

func RecordGnetHTTP1ChunkedFlushFallback() {
	gnetHTTP1.chunkedFallback.Add(1)
	gnetHTTP1.chunkedFlush.Add(1)
}

func RecordGnetHTTP1ChunkedHeaderFallback() {
	gnetHTTP1.chunkedFallback.Add(1)
	gnetHTTP1.chunkedHeader.Add(1)
}

func RecordGnetHTTP1DirectWriteAfterFlush() {
	gnetHTTP1.directWriteAfterFlush.Add(1)
}

func RecordGnetHTTP1AsyncWrite() {
	gnetHTTP1.asyncWrite.Add(1)
}

func SnapshotGnetHTTP1() GnetHTTP1Snapshot {
	s := GnetHTTP1Snapshot{
		HTTP1ClassifiedTotal:    gnetHTTP1.http1Classified.Load(),
		H2CClassifiedTotal:      gnetHTTP1.h2cClassified.Load(),
		SidecarClassifiedTotal:  gnetHTTP1.sidecarClassified.Load(),
		ClassifyErrorTotal:      gnetHTTP1.classifyError.Load(),
		EnqueueDropTotal:        gnetHTTP1.enqueueDrop.Load(),
		HTTP1EnqueueDropTotal:   gnetHTTP1.http1EnqueueDrop.Load(),
		H2CEnqueueDropTotal:     gnetHTTP1.h2cEnqueueDrop.Load(),
		SidecarEnqueueDropTotal: gnetHTTP1.sidecarEnqueueDrop.Load(),
		ResponseTotal:           gnetHTTP1.responseTotal.Load(),
		BufferedResponseTotal:   gnetHTTP1.bufferedResponse.Load(),
		InlineBufferHitTotal:    gnetHTTP1.inlineBufferHit.Load(),
		HeapBufferSpillTotal:    gnetHTTP1.heapBufferSpill.Load(),
		ContentLengthAutoTotal:  gnetHTTP1.contentLengthAuto.Load(),
		ChunkedFallbackTotal:    gnetHTTP1.chunkedFallback.Load(),
		ChunkedFlushTotal:       gnetHTTP1.chunkedFlush.Load(),
		ChunkedHeaderTotal:      gnetHTTP1.chunkedHeader.Load(),
		DirectWriteAfterFlush:   gnetHTTP1.directWriteAfterFlush.Load(),
		AsyncWriteTotal:         gnetHTTP1.asyncWrite.Load(),
	}
	s.BufferedResponseRatio = safeRatio(s.BufferedResponseTotal, s.ResponseTotal)
	s.InlineBufferHitRatio = safeRatio(s.InlineBufferHitTotal, s.BufferedResponseTotal)
	s.ChunkedFallbackRatio = safeRatio(s.ChunkedFallbackTotal, s.ResponseTotal)
	classifiedTotal := s.HTTP1ClassifiedTotal + s.H2CClassifiedTotal + s.SidecarClassifiedTotal
	s.HTTP1ClassificationRatio = safeRatio(s.HTTP1ClassifiedTotal, classifiedTotal)
	return s
}

func safeRatio(numerator int64, denominator int64) float64 {
	if denominator <= 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}
