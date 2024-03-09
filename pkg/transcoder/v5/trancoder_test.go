package v5

import (
	"context"
	"fmt"
	"testing"
	"transcode/pkg/config"
	"transcode/pkg/ffprobe"
	"transcode/pkg/request"
	"transcode/pkg/resolution"

	"github.com/stretchr/testify/assert"
	"github.com/thnthien/great-deku/container"
	"github.com/thnthien/great-deku/l"
)

func TestTranscoder_Transcode(t *testing.T) {
	serverConfig := config.ServerConfig{
		FfmpegBin:              "/usr/bin/ffmpeg",
		FfprobeBin:             "/usr/bin/ffprobe",
		OutputPath:             "~/Downloads/output",
		ClearAfterStream:       true,
		TranscoderVersion:      3,
		IgnoreBitrateThreshold: 153600,
	}
	container.NamedSingleton("ll", func() l.Logger {
		return l.New()
	})
	container.NamedSingleton("ffprobe", func() *ffprobe.Ffprobe {
		return ffprobe.New(serverConfig)
	})
	container.NamedSingleton("commandBuilder", func() *CommandBuilder {
		return NewCommandBuilder(serverConfig)
	})

	tr := New(serverConfig, request.TranscodeReq{
		FilePath:         "/home/thienthn/Downloads/1666689291478.mp4",
		StoredFolderPath: "/home/thienthn/Downloads/output/test",
		Resolutions:      []resolution.Resolution{resolution.R1080, resolution.R720, resolution.R360},
	})
	data, err := tr.Transcode(context.Background())
	assert.NoError(t, err)
	fmt.Printf("%+v", data)
}

func Test_DownBitRateValue(t *testing.T) {
	bitRate := defaultCommandBuilder.downBitRateValue(18000*1000, resolution.R1080, resolution.R720)
	assert.Equal(t, int64(2330168), bitRate)

	bitRate = defaultCommandBuilder.downBitRateValue(18000*1000, resolution.R1080, resolution.R1080)
	assert.Equal(t, int64(4*1024*1024), bitRate)

	bitRate = defaultCommandBuilder.downBitRateValue(18000*1000, resolution.R1080, resolution.R360)
	assert.Equal(t, int64(809085), bitRate)
}

func Test_BuildCommand(t *testing.T) {
	args, _ := defaultCommandBuilder.buildCommand(CommandConfig{
		FolderName:         "thienthn",
		FilePath:           "/home/thienthn/Downloads/190301_1_25_11.mp4",
		StoredFolderPath:   "/home/thienthn/Downloads/output/test",
		TargetResolutions:  []resolution.Resolution{resolution.R1080, resolution.R720, resolution.R360},
		SourceWidth:        1920,
		SourceHeight:       1080,
		SourceResolution:   1080,
		SourceDuration:     37,
		SourceBitRate:      18375447,
		SourceAudioBitRate: 317375,
		SourceFrameRate:    25,
	})
	assert.Equal(t, []string{"-y", "-threads", "1", "-hwaccel", "cuda", "-hwaccel_output_format", "cuda",
		"-i", "/home/thienthn/Downloads/190301_1_25_11.mp4", "-preset", "medium", "-c:v", "h264_nvenc",
		"-no-scenecut", "1", "-forced-idr", "1", "-force_key_frames", "expr:gte(t,n_forced*6)", "-ac", "2",
		"-map", "v:0", "-map", "v:0", "-map", "v:0", "-map", "a:0", "-map", "a:0", "-map", "a:0",
		"-filter:v:0", "scale_npp=-2:1080", "-b:v:0", "3584k", "-maxrate:v:0", "5376k", "-bufsize:v:0", "5376k",
		"-filter:v:1", "scale_npp=-2:720", "-b:v:1", "1991k", "-maxrate:v:1", "2986k", "-bufsize:v:1", "2986k",
		"-filter:v:2", "scale_npp=-2:360", "-b:v:2", "691k", "-maxrate:v:2", "1037k", "-bufsize:v:2", "1037k",
		"-b:a:0", "256k", "-b:a:1", "192k", "-b:a:2", "96k", "-f", "hls", "-hls_time", "6", "-hls_playlist_type", "vod",
		"-hls_flags", "independent_segments", "-hls_segment_type", "mpegts",
		"-hls_segment_filename", "/home/thienthn/Downloads/output/test/stream_%v/data%02d.ts",
		"-master_pl_name", "master.m3u8", "-var_stream_map", "v:0,a:0 v:1,a:1 v:2,a:2", "-fps_mode", "passthrough",
		"/home/thienthn/Downloads/output/test/stream_%v.m3u8"}, args)

	args, _ = defaultCommandBuilder.buildCommand(CommandConfig{
		FolderName:         "thienthn",
		FilePath:           "/home/thienthn/Downloads/1651904407363.mp4",
		StoredFolderPath:   "/home/thienthn/Downloads/output/test",
		TargetResolutions:  []resolution.Resolution{resolution.R1080, resolution.R720, resolution.R360},
		SourceWidth:        1920,
		SourceHeight:       1080,
		SourceResolution:   1080,
		SourceDuration:     1054,
		SourceBitRate:      491882,
		SourceAudioBitRate: 170658,
		SourceFrameRate:    30,
	})
	assert.Equal(t, []string{"-y", "-threads", "1", "-hwaccel", "cuda", "-hwaccel_output_format", "cuda",
		"-i", "/home/thienthn/Downloads/1651904407363.mp4", "-preset", "medium",
		"-no-scenecut", "1", "-forced-idr", "1", "-force_key_frames", "source",
		"-ac", "2", "-map", "v:0", "-map", "v:0", "-map", "a:0", "-map", "a:0", "-c:v:0", "copy", "-c:a:0", "copy", "-c:v:1", "h264_nvenc",
		"-filter:v:1", "scale_npp=-2:720", "-b:v:1", "266k", "-maxrate:v:1", "400k", "-bufsize:v:1", "400k", "-c:a:0", "copy", "-c:a:1", "copy",
		"-f", "hls", "-hls_time", "8.666666", "-hls_playlist_type", "vod", "-hls_flags", "independent_segments",
		"-hls_segment_type", "mpegts", "-hls_segment_filename", "/home/thienthn/Downloads/output/test/stream_%v/data%02d.ts",
		"-master_pl_name", "master.m3u8", "-var_stream_map", "v:0,a:0 v:1,a:1", "-fps_mode", "passthrough",
		"/home/thienthn/Downloads/output/test/stream_%v.m3u8"}, args)

	args, _ = defaultCommandBuilder.buildCommand(CommandConfig{
		FolderName:         "thienthn",
		FilePath:           "/home/thienthn/Downloads/hotkids.mp4",
		StoredFolderPath:   "/home/thienthn/Downloads/output/test",
		TargetResolutions:  []resolution.Resolution{resolution.R1080, resolution.R720, resolution.R360},
		SourceWidth:        1920,
		SourceHeight:       1080,
		SourceResolution:   1080,
		SourceDuration:     527,
		SourceBitRate:      1492330,
		SourceAudioBitRate: 317375,
		SourceFrameRate:    60,
	})
	assert.Equal(t, []string{"-y", "-threads", "1", "-hwaccel", "cuda", "-hwaccel_output_format", "cuda",
		"-i", "/home/thienthn/Downloads/hotkids.mp4", "-preset", "medium", "-c:v", "h264_nvenc",
		"-no-scenecut", "1", "-forced-idr", "1", "-force_key_frames", "expr:gte(t,n_forced*6)",
		"-ac", "2", "-map", "v:0", "-map", "v:0", "-map", "v:0", "-map", "a:0", "-map", "a:0", "-map", "a:0",
		"-filter:v:0", "scale_npp=-2:1080", "-b:v:0", "1457k", "-maxrate:v:0", "2186k", "-bufsize:v:0", "2186k",
		"-filter:v:1", "fps=30,scale_npp=-2:720", "-b:v:1", "809k", "-maxrate:v:1", "1214k", "-bufsize:v:1", "1214k",
		"-filter:v:2", "fps=30,scale_npp=-2:360", "-b:v:2", "187k", "-maxrate:v:2", "281k", "-bufsize:v:2", "281k",
		"-b:a:0", "256k", "-b:a:1", "192k", "-b:a:2", "96k", "-f", "hls", "-hls_time", "6", "-hls_playlist_type", "vod",
		"-hls_flags", "independent_segments", "-hls_segment_type", "mpegts",
		"-hls_segment_filename", "/home/thienthn/Downloads/output/test/stream_%v/data%02d.ts",
		"-master_pl_name", "master.m3u8", "-var_stream_map", "v:0,a:0 v:1,a:1 v:2,a:2", "-fps_mode", "passthrough",
		"/home/thienthn/Downloads/output/test/stream_%v.m3u8"}, args)
}
