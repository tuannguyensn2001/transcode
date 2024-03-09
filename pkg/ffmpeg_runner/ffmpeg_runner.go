package ffmpegrunner

import (
	"regexp"
	"strings"
	"transcode/pkg/commander"
)

var equalAndSpaceRe = regexp.MustCompile(`=\s+`)

type FfmpegRunner struct {
	commander.Commander
}

func New(ffmpegBin, ffprobeBin string) *FfmpegRunner {
	return &FfmpegRunner{
		Commander: commander.New(ffmpegBin),
	}
}

func (r *FfmpegRunner) Run() chan error {
	return r.Commander.Run()
}

func (r *FfmpegRunner) Logs() chan IProgress {
	out := make(chan IProgress)
	ls := r.Commander.StderrLogs()

	go func() {
		defer close(out)

		for line := range ls {
			switch progressDetectType(line) {
			case Frame:
				out <- r.frameProgress(line)
			case OpeningFile:
				out <- r.openingFileProgress(line)
			default:
				out <- NewProgress(line)
			}
		}
	}()

	return out
}

func (r *FfmpegRunner) frameProgress(line string) *FrameProgress {
	st := equalAndSpaceRe.ReplaceAllString(line, `=`)

	p := NewFrameProgress()
	f := strings.Fields(st)
	var framesProcessed string
	var currentTime string
	var currentBitrate string
	var currentSpeed string
	var fps string

	for j := 0; j < len(f); j++ {
		field := f[j]
		fieldSplit := strings.Split(field, "=")

		if len(fieldSplit) > 1 {
			fieldname := strings.Split(field, "=")[0]
			fieldvalue := strings.Split(field, "=")[1]

			if fieldname == "frame" {
				framesProcessed = fieldvalue
			}

			if fieldname == "time" {
				currentTime = fieldvalue
			}

			if fieldname == "bitrate" {
				currentBitrate = fieldvalue
			}
			if fieldname == "speed" {
				currentSpeed = fieldvalue
			}
			if fieldname == "fps" {
				fps = fieldvalue
			}
		}
	}

	p.CurrentBitrate = currentBitrate
	p.FramesProcessed = framesProcessed
	p.CurrentTime = currentTime
	p.Speed = currentSpeed
	p.FPS = fps
	return p
}

func (r *FfmpegRunner) openingFileProgress(line string) *OpeningFileProgress {
	st := equalAndSpaceRe.ReplaceAllString(line, `=`)
	p := NewOpeningFileProgress()
	f := strings.Fields(st)

	for i := range f {
		fieldSplit := strings.Split(f[i], "=")
		if len(fieldSplit) > 1 {
			name := fieldSplit[0]
			value := fieldSplit[1]

			switch name {
			case "bitrate":
				p.Bitrate = value
			case "speed":
				p.Speed = value
			}
		} else {
			value := fieldSplit[0]
			if strings.Contains(value, "'") {
				p.FilePath = strings.Replace(value[1:len(value)-1], "crypto:", "", -1)
			}
		}
	}

	return p
}

func progressDetectType(line string) ProgressType {
	if strings.Contains(line, "frame=") && strings.Contains(line, "time=") && strings.Contains(line, "bitrate=") {
		return Frame
	}
	if strings.Contains(line, "Opening") && strings.Contains(line, "for writing") {
		return OpeningFile
	}
	return ""
}
