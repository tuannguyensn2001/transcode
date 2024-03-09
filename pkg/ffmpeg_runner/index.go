package ffmpegrunner

import (
	"fmt"
)

type ProgressType string

const (
	Raw         ProgressType = "raw"
	Frame       ProgressType = "frame"
	OpeningFile ProgressType = "opening_file"
)

type IProgress interface {
	ToString() string
	GetType() ProgressType
}

type Progress struct {
	progressType ProgressType
	Raw          string
}

func (p *Progress) ToString() string {
	return p.Raw
}

func (p *Progress) GetType() ProgressType {
	return p.progressType
}

func NewProgress(line string) *Progress {
	return &Progress{progressType: Raw, Raw: line}
}

type FrameProgress struct {
	progressType    ProgressType
	FramesProcessed string
	CurrentTime     string
	CurrentBitrate  string
	Progress        float64
	Speed           string
	FPS             string
}

func NewFrameProgress() *FrameProgress {
	return &FrameProgress{progressType: Frame}
}

func (p *FrameProgress) ToString() string {
	return "frame=" + p.FramesProcessed + " time=" + p.CurrentTime + " bitrate=" + p.CurrentBitrate +
		" progress=" + fmt.Sprintf("%f", p.Progress) + " speed=" + p.Speed
}

func (p *FrameProgress) GetType() ProgressType {
	return p.progressType
}

type OpeningFileProgress struct {
	progressType ProgressType
	FilePath     string
	WritingRate  string
	Bitrate      string
	Speed        string
}

func (p *OpeningFileProgress) ToString() string {
	str := "opening '" + p.FilePath + "' for writing"
	if p.WritingRate != "" {
		str += "rate=" + p.WritingRate
	}
	str += " speed" + p.Speed
	return str
}

func (p *OpeningFileProgress) GetType() ProgressType {
	return p.progressType
}

func NewOpeningFileProgress() *OpeningFileProgress {
	return &OpeningFileProgress{progressType: OpeningFile}
}
