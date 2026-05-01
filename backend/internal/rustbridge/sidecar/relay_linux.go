//go:build linux

package sidecar

import (
	"io"
	"net"
	"syscall"

	"golang.org/x/sys/unix"
)

const relaySpliceChunk = 1 << 20

func relayCopyPlatform(dst net.Conn, src net.Conn) (int64, bool, error) {
	srcRaw, ok := relayRawConn(src)
	if !ok {
		return 0, false, nil
	}
	dstRaw, ok := relayRawConn(dst)
	if !ok {
		return 0, false, nil
	}

	pipeFDs := []int{0, 0}
	if err := unix.Pipe2(pipeFDs, unix.O_CLOEXEC|unix.O_NONBLOCK); err != nil {
		return 0, false, nil
	}
	defer unix.Close(pipeFDs[0])
	defer unix.Close(pipeFDs[1])
	_, _ = unix.FcntlInt(uintptr(pipeFDs[0]), unix.F_SETPIPE_SZ, relaySpliceChunk)

	var copied int64
	for {
		loaded, readErr := spliceConnToPipe(srcRaw, pipeFDs[1])
		if loaded > 0 {
			written, writeErr := splicePipeToConn(dstRaw, pipeFDs[0], loaded)
			copied += written
			if writeErr != nil {
				remaining := loaded - written
				if remaining > 0 && isSpliceFallbackError(writeErr) {
					drained, drainErr := drainPipeToConn(dst, pipeFDs[0], remaining)
					copied += drained
					if drainErr != nil {
						return copied, true, drainErr
					}
					return copied, false, nil
				}
				if isSpliceFallbackError(writeErr) {
					return copied, false, nil
				}
				return copied, true, writeErr
			}
		}
		if readErr != nil {
			if isSpliceFallbackError(readErr) {
				return copied, false, nil
			}
			return copied, true, readErr
		}
		if loaded == 0 {
			return copied, true, nil
		}
	}
}

func relayRawConn(conn net.Conn) (syscall.RawConn, bool) {
	sysConn, ok := conn.(syscall.Conn)
	if !ok {
		return nil, false
	}
	rawConn, err := sysConn.SyscallConn()
	if err != nil {
		return nil, false
	}
	return rawConn, true
}

func spliceConnToPipe(src syscall.RawConn, pipeWriter int) (int64, error) {
	var copied int64
	var spliceErr error
	if err := src.Read(func(fd uintptr) bool {
		n, err := unix.Splice(int(fd), nil, pipeWriter, nil, relaySpliceChunk, unix.SPLICE_F_MOVE|unix.SPLICE_F_NONBLOCK)
		copied = int64(n)
		spliceErr = err
		if n > 0 {
			spliceErr = nil
			return true
		}
		return !isSpliceRetryError(err)
	}); err != nil {
		return 0, err
	}
	return copied, spliceErr
}

func splicePipeToConn(dst syscall.RawConn, pipeReader int, size int64) (int64, error) {
	var copied int64
	for copied < size {
		chunk := int(size - copied)
		if chunk > relaySpliceChunk {
			chunk = relaySpliceChunk
		}
		var wrote int64
		var spliceErr error
		if err := dst.Write(func(fd uintptr) bool {
			n, err := unix.Splice(pipeReader, nil, int(fd), nil, chunk, unix.SPLICE_F_MOVE|unix.SPLICE_F_NONBLOCK)
			wrote = int64(n)
			spliceErr = err
			if n > 0 {
				spliceErr = nil
				return true
			}
			if err == nil && n == 0 {
				spliceErr = io.ErrNoProgress
				return true
			}
			return !isSpliceRetryError(err)
		}); err != nil {
			return copied, err
		}
		if spliceErr != nil {
			return copied, spliceErr
		}
		copied += wrote
	}
	return copied, nil
}

func drainPipeToConn(dst net.Conn, pipeReader int, size int64) (int64, error) {
	bufPtr := relayBufferPool.Get().(*[]byte)
	defer relayBufferPool.Put(bufPtr)
	buf := *bufPtr

	var copied int64
	for copied < size {
		limit := int(size - copied)
		if limit > len(buf) {
			limit = len(buf)
		}
		n, err := unix.Read(pipeReader, buf[:limit])
		if n > 0 {
			written, writeErr := writeFull(dst, buf[:n])
			copied += int64(written)
			if writeErr != nil {
				return copied, writeErr
			}
		}
		if err != nil {
			if err == unix.EINTR {
				continue
			}
			return copied, err
		}
		if n == 0 {
			return copied, io.ErrUnexpectedEOF
		}
	}
	return copied, nil
}

func writeFull(dst net.Conn, data []byte) (int, error) {
	var total int
	for total < len(data) {
		n, err := dst.Write(data[total:])
		total += n
		if err != nil {
			return total, err
		}
		if n == 0 {
			return total, io.ErrNoProgress
		}
	}
	return total, nil
}

func isSpliceRetryError(err error) bool {
	return err == unix.EAGAIN || err == unix.EWOULDBLOCK || err == unix.EINTR
}

func isSpliceFallbackError(err error) bool {
	return err == unix.EINVAL ||
		err == unix.ENOSYS ||
		err == unix.EOPNOTSUPP ||
		err == unix.ENOTSUP ||
		err == unix.EXDEV ||
		err == unix.EPERM
}
