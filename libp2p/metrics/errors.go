package metrics

import "errors"

// ErrInvalidValueForTimeToLiveParam signals that an invalid value for the time-to-live parameter was provided
var ErrInvalidValueForTimeToLiveParam = errors.New("invalid value for the time-to-live parameter")
