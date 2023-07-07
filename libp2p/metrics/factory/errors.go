package factory

import "errors"

// ErrUnknownConnectionWatcherType signals that an unknown connection watcher type was provided
var ErrUnknownConnectionWatcherType = errors.New("unknown connection type")
