package loop

// cmdProcessor groups the fields used for control-command deduplication
// and sequencing.
type cmdProcessor struct {
	processedIDs   map[string]struct{}
	cmdSeqCounter  uint64
	lastAppliedSeq uint64
}
