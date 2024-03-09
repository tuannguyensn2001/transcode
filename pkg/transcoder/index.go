package transcoder

import (
	"context"
	"transcode/pkg/resolution"
)

type UploadFile struct {
	Name      string // name of ts file
	Path      string // path to ts file
	UploadKey string // upload path on storage
}

func (f UploadFile) String() string {
	return "name=" + f.Name + " " + "Path=" + f.Path + " " + "UploadKey=" + f.UploadKey
}

type OutputData struct {
	Width             int
	Resolution        int
	FPS               int
	Duration          int
	VideoBitrate      int
	AudioBitrate      int
	TranscodeDuration int
	Resolutions       []resolution.Resolution
}

type ITranscoder interface {
	Transcode(ctx context.Context) (OutputData, error)
	Stop(isPause bool) error
	Output() chan UploadFile
}
