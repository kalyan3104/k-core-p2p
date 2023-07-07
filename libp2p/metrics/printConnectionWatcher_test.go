package metrics_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/kalyan3104/k-core-p2p/libp2p/metrics"
	"github.com/kalyan3104/k-core/core"
	"github.com/kalyan3104/k-core/core/check"
	"github.com/stretchr/testify/assert"
)

func TestNewPrintConnectionsWatcher(t *testing.T) {
	t.Parallel()

	t.Run("invalid value for time to live parameter should error", func(t *testing.T) {
		t.Parallel()

		pcw, err := metrics.NewPrintConnectionsWatcher(metrics.MinTimeToLive - time.Nanosecond)
		assert.True(t, check.IfNil(pcw))
		assert.True(t, errors.Is(err, metrics.ErrInvalidValueForTimeToLiveParam))
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		pcw, err := metrics.NewPrintConnectionsWatcher(metrics.MinTimeToLive)
		assert.False(t, check.IfNil(pcw))
		assert.Nil(t, err)

		_ = pcw.Close()
	})
}

func TestPrintConnectionsWatcher_Close(t *testing.T) {
	t.Parallel()

	t.Run("no iteration has been done", func(t *testing.T) {
		t.Parallel()

		pcw, _ := metrics.NewPrintConnectionsWatcher(time.Hour)
		err := pcw.Close()

		assert.Nil(t, err)
		time.Sleep(time.Second) // allow the go routine to close
		assert.True(t, pcw.GoRoutineClosed())
	})
	t.Run("iterations were done", func(t *testing.T) {
		t.Parallel()

		pcw, _ := metrics.NewPrintConnectionsWatcher(time.Second)
		time.Sleep(time.Second * 4)
		err := pcw.Close()

		assert.Nil(t, err)
		time.Sleep(time.Second) // allow the go routine to close
		assert.True(t, pcw.GoRoutineClosed())
	})

}

func TestPrintConnectionsWatcher_NewKnownConnection(t *testing.T) {
	t.Parallel()

	t.Run("invalid connection", func(t *testing.T) {
		providedPid := core.PeerID("pid")
		connection := " "
		numCalled := 0

		handler := func(pid core.PeerID, conn string) {
			numCalled++
		}
		pcw, _ := metrics.NewPrintConnectionsWatcherWithHandler(time.Hour, handler)

		pcw.NewKnownConnection(providedPid, connection)
		assert.Equal(t, 0, numCalled)
	})
	t.Run("valid connection", func(t *testing.T) {
		providedPid := core.PeerID("pid")
		connection := "connection"
		numCalled := 0

		handler := func(pid core.PeerID, conn string) {
			numCalled++
			assert.Equal(t, providedPid, pid)
			assert.Equal(t, connection, conn)
		}
		pcw, _ := metrics.NewPrintConnectionsWatcherWithHandler(time.Hour, handler)

		pcw.NewKnownConnection(providedPid, connection)
		assert.Equal(t, 1, numCalled)
		pcw.NewKnownConnection(providedPid, connection)
		assert.Equal(t, 1, numCalled)
	})
}

func TestLogPrintHandler_shouldNotPanic(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r != nil {
			assert.Fail(t, fmt.Sprintf("should have not panic: %v", r))
		}
	}()

	metrics.LogPrintHandler("pid", "connection")
}
