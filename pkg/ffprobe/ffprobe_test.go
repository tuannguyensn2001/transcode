package ffprobe

import (
	"log"
	"testing"
	"transcode/pkg/config"

	"github.com/stretchr/testify/assert"
	"github.com/thnthien/great-deku/container"
	"github.com/thnthien/great-deku/l"
)

func TestFfprobe_InputInfo(t *testing.T) {
	container.NamedSingleton("ll", func() l.Logger {
		return l.New()
	})
	f := New(config.ServerConfig{
		FfmpegBin:  "/usr/bin/ffmpeg",
		FfprobeBin: "/usr/bin/ffprobe",
	})

	info, err := f.InputInfo("/home/thienthn/Downloads/test.mp4", 2)
	assert.NoError(t, err)
	log.Printf("%+v", info)
}

func TestFfprobe_ReadFrame(t *testing.T) {
	container.NamedSingleton("ll", func() l.Logger {
		return l.New()
	})

	f := New(config.ServerConfig{
		FfmpegBin:  "/usr/bin/ffmpeg",
		FfprobeBin: "/usr/bin/ffprobe",
	})

	r := f.ReadFrame("/home/thienthn/Downloads/test.mp4")
	done := r.Run()
	frames := r.Logs()
loop:
	for {
		select {
		case err := <-done:
			assert.NoError(t, err)
			log.Printf("finish")
			break loop
		case frame := <-frames:
			log.Printf("%+v", frame)
		}
	}
	for frame := range frames {
		log.Printf("%+v", frame)
	}
}
