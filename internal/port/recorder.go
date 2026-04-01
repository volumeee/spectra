package port

// Recorder captures browser sessions as video or frame sequences.
type Recorder interface {
	// Start begins recording for the given session ID.
	Start(sessionID string) error
	// Stop ends recording and returns the result.
	Stop(sessionID string) (*RecordingResult, error)
}

// RecordingResult contains the output of a recording session.
type RecordingResult struct {
	SessionID  string `json:"session_id"`
	Format     string `json:"format"`   // "screencast" | "frames" | "both"
	VideoData  string `json:"video,omitempty"`   // base64 WebP animated
	Frames     []string `json:"frames,omitempty"` // base64 PNG frames
	FrameCount int    `json:"frame_count"`
	DurationMs int64  `json:"duration_ms"`
}
