package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/kalyan3104/k-core/core"
	"github.com/kalyan3104/k-core-storage-go/timecache"
)

const MinTimeToLive = minTimeToLive

// NewPrintConnectionsWatcherWithHandler -
func NewPrintConnectionsWatcherWithHandler(timeToLive time.Duration, handler func(pid core.PeerID, connection string)) (*printConnectionsWatcher, error) {
	if timeToLive < minTimeToLive {
		return nil, fmt.Errorf("%w in NewPrintConnectionsWatcher, got: %d, minimum: %d", ErrInvalidValueForTimeToLiveParam, timeToLive, minTimeToLive)
	}

	pcw := &printConnectionsWatcher{
		timeToLive:   timeToLive,
		timeCacher:   timecache.NewTimeCache(timeToLive),
		printHandler: handler,
	}

	ctx, cancel := context.WithCancel(context.Background())
	pcw.cancel = cancel
	go pcw.doSweep(ctx)

	return pcw, nil
}

func LogPrintHandler(pid core.PeerID, connection string) {
	logPrintHandler(pid, connection)
}

func (pcw *printConnectionsWatcher) GoRoutineClosed() bool {
	return pcw.goRoutineClosed.IsSet()
}
