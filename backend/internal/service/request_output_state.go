package service

import (
	"sync"
	"sync/atomic"

	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
)

const gatewayRequestOutputStateContextKey = "gateway_request_output_state"

type gatewayResponseProtocol int32

const (
	gatewayResponseProtocolUnknown gatewayResponseProtocol = iota
	gatewayResponseProtocolOpenAISSE
	gatewayResponseProtocolAnthropicSSE
	gatewayResponseProtocolBufferedJSON
)

type requestOutputState struct {
	protocol      atomic.Int32
	protocolBegan atomic.Bool
	payloadBegan  atomic.Bool
}

type requestOutputStateHolder struct {
	mu    sync.Mutex
	state *requestOutputState
}

func getRequestOutputState(c gatewayctx.GatewayContext) *requestOutputState {
	if c == nil {
		return nil
	}
	if raw, ok := c.Value(gatewayRequestOutputStateContextKey); ok {
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
	c.SetValue(gatewayRequestOutputStateContextKey, holder)
	return holder.state
}

func markRequestProtocolStarted(c gatewayctx.GatewayContext, protocol gatewayResponseProtocol) {
	state := getRequestOutputState(c)
	if state == nil {
		return
	}
	if protocol != gatewayResponseProtocolUnknown {
		state.protocol.CompareAndSwap(int32(gatewayResponseProtocolUnknown), int32(protocol))
	}
	state.protocolBegan.Store(true)
}

func markRequestPayloadStarted(c gatewayctx.GatewayContext, protocol gatewayResponseProtocol) {
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

func requestPayloadStarted(c gatewayctx.GatewayContext) bool {
	state := getRequestOutputState(c)
	return state != nil && state.payloadBegan.Load()
}

func requestProtocolState(c gatewayctx.GatewayContext) (gatewayResponseProtocol, bool) {
	state := getRequestOutputState(c)
	if state == nil || !state.protocolBegan.Load() {
		return gatewayResponseProtocolUnknown, false
	}
	return gatewayResponseProtocol(state.protocol.Load()), true
}

func RequestPayloadStarted(c gatewayctx.GatewayContext) bool {
	return requestPayloadStarted(c)
}

func RequestUsesOpenAISSE(c gatewayctx.GatewayContext) bool {
	protocol, ok := requestProtocolState(c)
	return ok && protocol == gatewayResponseProtocolOpenAISSE
}

func RequestUsesAnthropicSSE(c gatewayctx.GatewayContext) bool {
	protocol, ok := requestProtocolState(c)
	return ok && protocol == gatewayResponseProtocolAnthropicSSE
}

func RequestUsesBufferedJSON(c gatewayctx.GatewayContext) bool {
	protocol, ok := requestProtocolState(c)
	return ok && protocol == gatewayResponseProtocolBufferedJSON
}

func MarkRequestOpenAISSEStarted(c gatewayctx.GatewayContext) {
	markRequestProtocolStarted(c, gatewayResponseProtocolOpenAISSE)
}

func MarkRequestAnthropicSSEStarted(c gatewayctx.GatewayContext) {
	markRequestProtocolStarted(c, gatewayResponseProtocolAnthropicSSE)
}

func MarkRequestBufferedJSONStarted(c gatewayctx.GatewayContext) {
	markRequestProtocolStarted(c, gatewayResponseProtocolBufferedJSON)
}

