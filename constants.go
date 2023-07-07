package p2p

import "time"

// NodeOperation defines the p2p node operation
type NodeOperation string

// NormalOperation defines the normal mode operation: either seeder, observer or validator
const NormalOperation NodeOperation = "normal operation"

// FullArchiveMode defines the node operation as a full archive mode
const FullArchiveMode NodeOperation = "full archive mode"

const (
	displayLastPidChars = 12

	// ListsSharder is the variant that uses lists
	ListsSharder = "ListsSharder"
	// OneListSharder is the variant that is shard agnostic and uses one list
	OneListSharder = "OneListSharder"
	// NilListSharder is the variant that will not do connection trimming
	NilListSharder = "NilListSharder"

	// ConnectionWatcherTypePrint - new connection found will be printed in the log file
	ConnectionWatcherTypePrint = "print"
	// ConnectionWatcherTypeDisabled - no connection watching should be made
	ConnectionWatcherTypeDisabled = "disabled"
	// ConnectionWatcherTypeEmpty - not set, no connection watching should be made
	ConnectionWatcherTypeEmpty = ""

	// WrongP2PMessageBlacklistDuration represents the time to keep a peer id in the blacklist if it sends a message that
	// do not follow this protocol
	WrongP2PMessageBlacklistDuration = time.Second * 7200
)
