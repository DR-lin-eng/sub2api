package service

import (
	"sync"
	"sync/atomic"

	"github.com/gin-gonic/gin"
)

const gatewayRequestOutputStateContextKey = "gateway_request_output_state"

type gatewayResponseProtocol int32

const (
	gatewayResponseProtocolUnknown gatewayResponseProtocol = iota
	gatewayResponseProtocolOpenAISSE
	gatewayResponseProtocolAnthropicSSE
	gatewayResponseProtocolBufferedJSON
)

// requestOutputState tracks whether we have only sent auxiliary keepalives/pings
// or committed real downstream payload bytes that make cross-account failover unsafe.
type requestOutputState struct {
	protocol      atomic.Int32
	protocolBegan atomic.Bool
	payloadBegan  atomic.Bool
}

type requestOutputStateHolder struct {
	mu    sync.Mutex
	state *requestOutputState
}

func getRequestOutputState(c *gin.Context) *requestOutputState {
	if c == nil {
		return nil
	}
	if raw, ok := c.Get(gatewayRequestOutputStateContextKey); ok {
		if state, ok := raw.(*requestOutputState); ok && state != nil {
			return state
		}
		if holder, ok := raw.(*requestOutputStateHolder); ok && holder != nil {
			holder.mu.Lock()
			defer holder.mu.Unlock()
			if holder.state == nil {
				holder.state = &requestOutputState{}
			}
			return holder.state
		}
	}
	holder := &requestOutputStateHolder{state: &requestOutputState{}}
	c.Set(gatewayRequestOutputStateContextKey, holder)
	return holder.state
}

func markRequestProtocolStarted(c *gin.Context, protocol gatewayResponseProtocol) {
	state := getRequestOutputState(c)
	if state == nil {
		return
	}
	if protocol != gatewayResponseProtocolUnknown {
		state.protocol.CompareAndSwap(int32(gatewayResponseProtocolUnknown), int32(protocol))
	}
	state.protocolBegan.Store(true)
}

func markRequestPayloadStarted(c *gin.Context, protocol gatewayResponseProtocol) {
	state := getRequestOutputState(c)
	if state == nil {
		return
	}
	if protocol != gatewayResponseProtocolUnknown {
		state.protocol.CompareAndSwap(int32(gatewayResponseProtocolUnknown), int32(protocol))
	}
	state.protocolBegan.Store(true)
	state.payloadBegan.Store(true)
}

func requestPayloadStarted(c *gin.Context) bool {
	state := getRequestOutputState(c)
	return state != nil && state.payloadBegan.Load()
}

func requestProtocolState(c *gin.Context) (gatewayResponseProtocol, bool) {
	state := getRequestOutputState(c)
	if state == nil || !state.protocolBegan.Load() {
		return gatewayResponseProtocolUnknown, false
	}
	return gatewayResponseProtocol(state.protocol.Load()), true
}

func RequestPayloadStarted(c *gin.Context) bool {
	return requestPayloadStarted(c)
}

func RequestUsesOpenAISSE(c *gin.Context) bool {
	protocol, ok := requestProtocolState(c)
	return ok && protocol == gatewayResponseProtocolOpenAISSE
}

func RequestUsesAnthropicSSE(c *gin.Context) bool {
	protocol, ok := requestProtocolState(c)
	return ok && protocol == gatewayResponseProtocolAnthropicSSE
}

func RequestUsesBufferedJSON(c *gin.Context) bool {
	protocol, ok := requestProtocolState(c)
	return ok && protocol == gatewayResponseProtocolBufferedJSON
}

func MarkRequestOpenAISSEStarted(c *gin.Context) {
	markRequestProtocolStarted(c, gatewayResponseProtocolOpenAISSE)
}

func MarkRequestAnthropicSSEStarted(c *gin.Context) {
	markRequestProtocolStarted(c, gatewayResponseProtocolAnthropicSSE)
}

func MarkRequestBufferedJSONStarted(c *gin.Context) {
	markRequestProtocolStarted(c, gatewayResponseProtocolBufferedJSON)
}
