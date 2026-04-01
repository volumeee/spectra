package recording

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image/png"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/spectra-browser/spectra/internal/port"
)

type session struct {
	page      *rod.Page
	frames    [][]byte
	startTime time.Time
	cancel    context.CancelFunc
	mu        sync.Mutex
}

// CDPRecorder captures browser frames via CDP Page.startScreencast events.
type CDPRecorder struct {
	format   string
	fps      int
	quality  int
	dir      string
	mu       sync.RWMutex
	sessions map[string]*session
}

func New(format string, fps, quality int, dir string) *CDPRecorder {
	if fps <= 0 {
		fps = 5
	}
	if quality <= 0 || quality > 100 {
		quality = 80
	}
	return &CDPRecorder{
		format:   format,
		fps:      fps,
		quality:  quality,
		dir:      dir,
		sessions: make(map[string]*session),
	}
}

// StartPage begins recording a rod.Page. sessionID must be unique per request.
func (r *CDPRecorder) StartPage(sessionID string, page *rod.Page) error {
	ctx, cancel := context.WithCancel(context.Background())

	s := &session{
		page:      page,
		startTime: time.Now(),
		cancel:    cancel,
	}

	r.mu.Lock()
	r.sessions[sessionID] = s
	r.mu.Unlock()

	everyNth := 30 / r.fps
	if everyNth < 1 {
		everyNth = 1
	}
	quality, maxW, maxH := r.quality, 1920, 1080

	err := proto.PageStartScreencast{
		Format:        proto.PageStartScreencastFormatPng,
		Quality:       &quality,
		MaxWidth:      &maxW,
		MaxHeight:     &maxH,
		EveryNthFrame: &everyNth,
	}.Call(page)
	if err != nil {
		cancel()
		return fmt.Errorf("start screencast: %w", err)
	}

	go func() {
		page.EachEvent(func(e *proto.PageScreencastFrame) {
			select {
			case <-ctx.Done():
				return
			default:
			}
			// e.Data is already []byte (base64-decoded by rod)
			s.mu.Lock()
			s.frames = append(s.frames, e.Data)
			s.mu.Unlock()
			_ = proto.PageScreencastFrameAck{SessionID: e.SessionID}.Call(page)
		})()
	}()

	slog.Debug("recording started", "session_id", sessionID)
	return nil
}

// Stop ends recording and returns the result.
func (r *CDPRecorder) Stop(sessionID string) (*port.RecordingResult, error) {
	r.mu.Lock()
	s, ok := r.sessions[sessionID]
	if ok {
		delete(r.sessions, sessionID)
	}
	r.mu.Unlock()

	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	s.cancel()
	_ = proto.PageStopScreencast{}.Call(s.page)

	s.mu.Lock()
	frames := s.frames
	s.mu.Unlock()

	duration := time.Since(s.startTime).Milliseconds()

	result := &port.RecordingResult{
		SessionID:  sessionID,
		Format:     r.format,
		FrameCount: len(frames),
		DurationMs: duration,
	}

	if r.format == "frames" || r.format == "both" {
		encoded := make([]string, len(frames))
		for i, f := range frames {
			encoded[i] = base64.StdEncoding.EncodeToString(f)
		}
		result.Frames = encoded
	}

	if (r.format == "screencast" || r.format == "both") && len(frames) > 0 {
		video, err := assemblePNGSequence(frames)
		if err != nil {
			slog.Warn("failed to assemble video", "error", err)
		} else {
			result.VideoData = base64.StdEncoding.EncodeToString(video)
			if r.dir != "" {
				_ = os.MkdirAll(r.dir, 0755)
				path := filepath.Join(r.dir, sessionID+".bin")
				_ = os.WriteFile(path, video, 0644)
			}
		}
	}

	slog.Debug("recording stopped", "session_id", sessionID, "frames", len(frames), "duration_ms", duration)
	return result, nil
}

// assemblePNGSequence concatenates PNG frames into a binary sequence.
// Each frame is prefixed with its 4-byte big-endian length.
// For true animated WebP/MP4, integrate golang.org/x/image or ffmpeg.
func assemblePNGSequence(frames [][]byte) ([]byte, error) {
	if len(frames) == 0 {
		return nil, fmt.Errorf("no frames")
	}
	// Validate first frame is valid PNG
	if _, err := png.Decode(bytes.NewReader(frames[0])); err != nil {
		return nil, fmt.Errorf("invalid PNG frame: %w", err)
	}
	var buf bytes.Buffer
	for _, frame := range frames {
		l := len(frame)
		buf.WriteByte(byte(l >> 24))
		buf.WriteByte(byte(l >> 16))
		buf.WriteByte(byte(l >> 8))
		buf.WriteByte(byte(l))
		buf.Write(frame)
	}
	return buf.Bytes(), nil
}
