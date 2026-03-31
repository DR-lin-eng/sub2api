//go:build !cgo || (!linux && !darwin)

package ffi

type dynamicLibrary struct {
	path string
}

func loadDynamicLibrary(path string) (*dynamicLibrary, error) {
	return &dynamicLibrary{path: path}, nil
}

func (d *dynamicLibrary) Close() error {
	return nil
}

func (d *dynamicLibrary) CallSHA256Hex(input []byte) (string, bool) {
	return "", false
}

func (d *dynamicLibrary) CallBuildSSEDataFrame(input []byte) ([]byte, bool) {
	return nil, false
}

func (d *dynamicLibrary) CallCorrectOpenAIToolCalls(input []byte) ([]byte, bool) {
	return nil, false
}

func (d *dynamicLibrary) CallRewriteOpenAIWSMessageForClient(input []byte, fromModel, toModel string, applyToolCorrection bool) ([]byte, bool) {
	return nil, false
}

func (d *dynamicLibrary) CallRewriteOpenAISSELineForClient(input []byte, fromModel, toModel string, applyToolCorrection bool) ([]byte, bool) {
	return nil, false
}

func (d *dynamicLibrary) CallRewriteOpenAISSEBodyForClient(input []byte, fromModel, toModel string, applyToolCorrection bool) ([]byte, bool) {
	return nil, false
}

func (d *dynamicLibrary) CallRewriteOpenAIWSMessageToSSEFrameForClient(input []byte, fromModel, toModel string, applyToolCorrection bool) ([]byte, bool) {
	return nil, false
}

func (d *dynamicLibrary) CallOpenAIWSParseUsage(input []byte) (int, int, int, bool) {
	return 0, 0, 0, false
}

func (d *dynamicLibrary) CallOpenAIWSParseEnvelope(input []byte) (string, string, string, bool) {
	return "", "", "", false
}

func (d *dynamicLibrary) CallOpenAIWSParseErrorFields(input []byte) (string, string, string, bool) {
	return "", "", "", false
}

func (d *dynamicLibrary) CallOpenAIWSParseRequestPayloadSummary(input []byte) (string, string, string, string, bool, bool, bool, bool) {
	return "", "", "", "", false, false, false, false
}

func (d *dynamicLibrary) CallOpenAIWSParseFrameSummary(input []byte) (string, string, string, string, string, string, int, int, int, bool, bool, bool, bool) {
	return "", "", "", "", "", "", 0, 0, 0, false, false, false, false
}

func (d *dynamicLibrary) CallParseOpenAISSEBodySummary(input []byte) (string, string, string, int, int, int, bool, bool, bool) {
	return "", "", "", 0, 0, 0, false, false, false
}

func (d *dynamicLibrary) CallOpenAIWSIsTerminalEvent(eventType string) (bool, bool) {
	return false, false
}

func (d *dynamicLibrary) CallOpenAIWSIsTokenEvent(eventType string) (bool, bool) {
	return false, false
}

func (d *dynamicLibrary) CallOpenAIWSHasToolCalls(message []byte) (bool, bool) {
	return false, false
}

func (d *dynamicLibrary) CallOpenAIWSReplaceModel(input []byte, fromModel, toModel string) ([]byte, bool) {
	return nil, false
}

func (d *dynamicLibrary) CallOpenAIWSDropPreviousResponseID(input []byte) ([]byte, bool) {
	return nil, false
}

func (d *dynamicLibrary) CallOpenAIWSSetPreviousResponseID(input []byte, previousResponseID string) ([]byte, bool) {
	return nil, false
}

func (d *dynamicLibrary) CallOpenAIWSSetRequestType(input []byte, eventType string) ([]byte, bool) {
	return nil, false
}

func (d *dynamicLibrary) CallOpenAIWSSetTurnMetadata(input []byte, turnMetadata string) ([]byte, bool) {
	return nil, false
}

func (d *dynamicLibrary) CallOpenAIWSSetInputSequence(input []byte, inputSequenceJSON []byte) ([]byte, bool) {
	return nil, false
}

func (d *dynamicLibrary) CallOpenAIWSNormalizePayloadWithoutInputAndPreviousResponseID(input []byte) ([]byte, bool) {
	return nil, false
}

func (d *dynamicLibrary) CallOpenAIWSBuildReplayInputSequence(previousFullInputJSON []byte, previousFullInputExists bool, currentPayload []byte, hasPreviousResponseID bool) ([]byte, bool, bool) {
	return nil, false, false
}
