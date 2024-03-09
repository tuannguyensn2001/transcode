package ffprobe

import (
	"strconv"
	"strings"
	"transcode/pkg/commander"
)

func (f *Ffprobe) ReadFrame(input string) *ReadFramer {
	args := []string{
		"-show_frames", "-show_entries",
		"frame=key_frame,media_type,duration_time,pkt_size", input,
	}
	return &ReadFramer{
		Commander: commander.New(f.ffprobeBin, args...),
	}
}

type FrameMediaType string

const (
	Video FrameMediaType = "video"
	Audio FrameMediaType = "audio"
)

type Frame struct {
	MediaType    FrameMediaType `json:"media_type"`
	KeyFrame     int            `json:"key_frame"`
	DurationTime float64        `json:"duration_time"`
	PktSize      int64          `json:"pkt_size"`
}

type ReadFramer struct {
	commander.Commander
}

func (r *ReadFramer) Logs() chan Frame {
	frames := make(chan Frame)
	go r.handleLogs(frames)

	return frames
}

func (r *ReadFramer) handleLogs(frames chan Frame) {
	defer close(frames)
	ls := r.Commander.StdoutLogs()

	var f Frame
	for line := range ls {
		if line == "[FRAME]" {
			f.DurationTime = 0
			f.MediaType = ""
			f.KeyFrame = 0
			f.PktSize = 0
		} else if line == "[/FRAME]" {
			frames <- f
		} else {
			value := getValue(line)
			if strings.HasPrefix(line, "media_type") {
				f.MediaType = FrameMediaType(value)
			} else if strings.HasPrefix(line, "key_frame") {
				val, _ := strconv.Atoi(value)
				f.KeyFrame = val
			} else if strings.HasPrefix(line, "duration_time") {
				val, _ := strconv.ParseFloat(value, 64)
				f.DurationTime = val
			} else if strings.HasPrefix(line, "pkt_size") {
				val, _ := strconv.ParseInt(value, 10, 64)
				f.PktSize = val
			}
		}
	}
}
