package server

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOptimizedListenerFileDelegatesToUnderlyingListener(t *testing.T) {
	base, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer base.Close()

	ln := &optimizedListener{Listener: base}
	file, err := ln.File()
	require.NoError(t, err)
	require.NotNil(t, file)
	_ = file.Close()
}
