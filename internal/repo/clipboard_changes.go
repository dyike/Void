package repo

// The caller serializes snapshots. A copy made during a read is retried on the
// next tick rather than recording a snapshot assembled from two versions.
type clipboardReadState struct {
	version int64
	seen    bool
}

func (s *clipboardReadState) read(version func() int64, snapshot func() (string, string)) (string, string) {
	before := version()
	if s.seen && s.version == before {
		return "", ""
	}
	content, kind := snapshot()
	if version() != before {
		return "", ""
	}
	s.version, s.seen = before, true
	return content, kind
}
