package v5

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"transcode/pkg/config"
	"transcode/pkg/resolution"
)

var (
	Kb       = int64(1024)
	Mb       = Kb * Kb
	tmpVideo = []string{"-map", "v:0"}
	tmpAudio = []string{"-map", "a:0"}
)

var defaultCommandBuilder CommandBuilder

func init() {
	defaultCommandBuilder.df1080Bitrate = Mb * 35 / 10                                 // 3.5Mb
	defaultCommandBuilder.df720Bitrate = defaultCommandBuilder.df1080Bitrate * 10 / 18 // df1080 / 1.8, 2.2Mb
	defaultCommandBuilder.df480Bitrate = defaultCommandBuilder.df720Bitrate * 10 / 18  // df720 / 1.8, 1.2Mb
	defaultCommandBuilder.df360Bitrate = defaultCommandBuilder.df480Bitrate * 10 / 16  // df480 / 1.6, 790Kb
	defaultCommandBuilder.df1080AudioBitrate = 256 * Kb
	defaultCommandBuilder.df720AudioBitrate = 192 * Kb
	defaultCommandBuilder.df480AudioBitrate = 128 * Kb
	defaultCommandBuilder.df360AudioBitrate = 96 * Kb
	defaultCommandBuilder.defaultBitrate = map[resolution.Resolution]defaultBitRate{
		resolution.R1080: {defaultCommandBuilder.df1080Bitrate, defaultCommandBuilder.df1080AudioBitrate},
		resolution.R720:  {defaultCommandBuilder.df720Bitrate, defaultCommandBuilder.df720AudioBitrate},
		resolution.R480:  {defaultCommandBuilder.df480Bitrate, defaultCommandBuilder.df480AudioBitrate},
		resolution.R360:  {defaultCommandBuilder.df360Bitrate, defaultCommandBuilder.df360AudioBitrate},
	}
	defaultCommandBuilder.ignoreResolutionThreshold = 150 * Kb
	defaultCommandBuilder.frameRateThreshold = 48
	defaultCommandBuilder.targetDuration = 6
}

type CommandBuilder struct {
	df1080Bitrate             int64
	df720Bitrate              int64
	df480Bitrate              int64
	df360Bitrate              int64
	df1080AudioBitrate        int64
	df720AudioBitrate         int64
	df480AudioBitrate         int64
	df360AudioBitrate         int64
	defaultBitrate            map[resolution.Resolution]defaultBitRate
	ignoreResolutionThreshold int64
	frameRateThreshold        int
	targetDuration            int
}

func NewCommandBuilder(cfg config.ServerConfig) *CommandBuilder {
	cb := defaultCommandBuilder
	if cfg.Default1080Bitrate > 0 {
		cb.df1080Bitrate = cfg.Default1080Bitrate
	}
	if cfg.IgnoreBitrateThreshold > 0 {
		cb.ignoreResolutionThreshold = cfg.IgnoreBitrateThreshold
	}
	if cfg.TargetSegmentDuration > 0 {
		cb.targetDuration = cfg.TargetSegmentDuration
	}
	return &cb
}

type defaultBitRate struct {
	Video int64
	Audio int64
}

type filterBitRate struct {
	inputBitRate string
	maxRate      string
}

type CommandConfig struct {
	FolderName         string                  `json:"folder_name"`
	FilePath           string                  `json:"file_path"`
	StoredFolderPath   string                  `json:"stored_folder_path"`
	KeyInfoFilePath    string                  `json:"key_info_file_path"`
	TargetResolutions  []resolution.Resolution `json:"target_resolutions"`
	SourceResolution   resolution.Resolution   `json:"source_resolution"`
	SourceWidth        int64                   `json:"width"`
	SourceHeight       int64                   `json:"height"`
	SourceDuration     int                     `json:"duration"`
	SourceBitRate      int64                   `json:"source_bit_rate"`
	SourceAudioBitRate int64                   `json:"source_audio_bit_rate"`
	SourceFrameRate    int                     `json:"source_frame_rate"`
}

func (b *CommandBuilder) downBitRateValue(bitRate int64, currentRes resolution.Resolution, targetRes resolution.Resolution) int64 {
	defBitRate := b.defaultBitrate[currentRes]
	if bitRate > defBitRate.Video {
		bitRate = defBitRate.Video
	}
	for currentRes.Level() > targetRes.Level() {
		switch currentRes {
		case resolution.R1080, resolution.R720:
			bitRate = bitRate * 10 / 18
		case resolution.R480:
			bitRate = bitRate * 10 / 16
		}
		currentRes = resolution.FromLevel(currentRes.Level() - 1)
	}

	return bitRate
}

func (b *CommandBuilder) buildCommand(cfg CommandConfig) ([]string, []resolution.Resolution) {
	m3u8Output := filepath.Join(cfg.StoredFolderPath, "stream_%v.m3u8")
	tsOutput := filepath.Join(cfg.StoredFolderPath, "stream_%v_data%02d.ts")

	cfg.TargetResolutions = b.chooseTargetResolutions(cfg)
	if len(cfg.TargetResolutions) == 0 {
		return nil, nil
	}
	var filterBitRates map[resolution.Resolution]filterBitRate
	filterBitRates, cfg.TargetResolutions = b.buildFilterBitRates(cfg)

	args := b.buildTranscodeCommand(cfg, filterBitRates, m3u8Output, tsOutput)
	return args, cfg.TargetResolutions
}

func (b *CommandBuilder) buildTranscodeCommand(cfg CommandConfig, bitRates map[resolution.Resolution]filterBitRate, m3u8Output, tsOutput string) []string {
	// example command:
	// ffmpeg -y -hwaccel cuda -hwaccel_output_format cuda -i rtmp://127.0.0.1:1935/live/7868802855338312
	// -preset medium -c:v h264_nvenc -no-scenecut 1 -forced-idr 1 -force_key_frames "expr:gte(t,n_forced*6)"
	// -ac 2 -map v:0 -map v:0 -map v:0 -map v:0 -map a:0 -map a:0 -map a:0 -map a:0
	// -filter:v:0 scale_npp=-2:1080 -b:v:0 5632k -maxrate:v:0 6195k -bufsize:v:0 6195k
	// -filter:v:1 scale_npp=-2:720 -b:v:1 3754k -maxrate:v:1 4130k -bufsize:v:1 4130k
	// -filter:v:2 scale_npp=-2:480 -b:v:2 2503k -maxrate:v:2 2753k -bufsize:v:2 2753k
	// -filter:v:3 scale_npp=-2:360 -b:v:3 1877k -maxrate:v:3 2065k -bufsize:v:3 2065k
	// -b:a:0 192k -b:a:1 192k -b:a:2 128k -b:a:3 128k
	// -f hls -hls_time 10 -hls_playlist_type vod
	// -hls_flags independent_segments -hls_segment_type mpegts
	// -hls_segment_filename output/53011690794520577/1678766701573/stream_%v/data%02d.ts
	// -master_pl_name master.m3u8
	// -var_stream_map "v:0,a:0 v:1,a:1 v:2,a:2 v:3,a:3"
	// -fps_mode passthrough output/53011690794520577/1678766701573/stream_%v.m3u8

	args := []string{
		"-y", "-threads", "1", "-hwaccel", "cuda", "-hwaccel_output_format", "cuda", "-i", cfg.FilePath,
		"-preset", "medium", "-c:v", "h264_nvenc", "-no-scenecut", "1", "-forced-idr", "1",
		"-force_key_frames", fmt.Sprintf("expr:gte(t,n_forced*%d)", b.targetDuration), "-ac", "2",
	}

	resLen := len(cfg.TargetResolutions)
	videoMap := make([]string, 0, resLen*2)
	audioMap := make([]string, 0, resLen*2)
	filterList := make([]string, 0, resLen*2)
	bitRateList := make([]string, 0, resLen*2)
	streamMap := make([]string, 0, resLen*2)

	scaleDownFrameRate := 0
	if cfg.SourceFrameRate >= b.frameRateThreshold {
		// if source fps >= fps threshold, minimize it by 2
		scaleDownFrameRate = cfg.SourceFrameRate / 2
	}
	for idx, res := range cfg.TargetResolutions {
		dbr := b.defaultBitrate[res]
		videoMap = append(videoMap, tmpVideo...)
		audioMap = append(audioMap, tmpAudio...)
		streamMap = append(streamMap, fmt.Sprintf("v:%d,a:%d", idx, idx))
		var filter, bitRate []string
		if cfg.SourceAudioBitRate > dbr.Audio {
			bitRate = []string{fmt.Sprintf("-b:a:%d", idx), fmt.Sprintf("%dk", dbr.Audio/Kb)}
		} else {
			bitRate = []string{fmt.Sprintf("-c:a:%d", idx), "copy"}
		}
		switch res {
		case resolution.R1080:
			filter = []string{
				fmt.Sprintf("-filter:v:%d", idx), "scale_npp=-2:1080",
			}
		case resolution.R720:
			val := "scale_npp=-2:720"
			if scaleDownFrameRate != 0 {
				val = fmt.Sprintf("fps=%d,", scaleDownFrameRate) + val
			}
			filter = []string{
				fmt.Sprintf("-filter:v:%d", idx), val,
			}
		case resolution.R480:
			val := "scale_npp=-2:480"
			if scaleDownFrameRate != 0 {
				val = fmt.Sprintf("fps=%d,", scaleDownFrameRate) + val
			}
			filter = []string{
				fmt.Sprintf("-filter:v:%d", idx), val,
			}
		case resolution.R360:
			val := "scale_npp=-2:360"
			if scaleDownFrameRate != 0 {
				val = fmt.Sprintf("fps=%d,", scaleDownFrameRate) + val
			}
			filter = []string{
				fmt.Sprintf("-filter:v:%d", idx), val,
			}
		default:
			val := fmt.Sprintf("scale_npp=-2:%d", res)
			if scaleDownFrameRate != 0 {
				val = fmt.Sprintf("fps=%d,", scaleDownFrameRate) + val
			}
			filter = []string{
				fmt.Sprintf("-filter:v:%d", idx), val,
			}
		}
		filter = append(filter, []string{
			fmt.Sprintf("-b:v:%d", idx), bitRates[res].inputBitRate,
			fmt.Sprintf("-maxrate:v:%d", idx), bitRates[res].maxRate,
			fmt.Sprintf("-bufsize:v:%d", idx), bitRates[res].maxRate,
		}...)
		filterList = append(filterList, filter...)
		bitRateList = append(bitRateList, bitRate...)
	}

	args = append(args, videoMap...)
	args = append(args, audioMap...)
	args = append(args, filterList...)
	args = append(args, bitRateList...)

	args = append(args, []string{
		"-f", "hls",
		"-hls_time", fmt.Sprintf("%d", b.targetDuration), "-hls_playlist_type", "vod", "-hls_flags", "independent_segments",
		"-hls_segment_type", "mpegts", "-hls_segment_filename", tsOutput,
	}...)
	if cfg.KeyInfoFilePath != "" {
		args = append(args, "-hls_key_info_file", cfg.KeyInfoFilePath)
	}
	args = append(args, "-master_pl_name", "master.m3u8", "-var_stream_map", strings.Join(streamMap, " "),
		"-fps_mode", "passthrough", m3u8Output,
	)

	return args
}

func (b *CommandBuilder) chooseTargetResolutions(cfg CommandConfig) []resolution.Resolution {
	resMap := make(map[resolution.Resolution]struct{})
	for _, r := range cfg.TargetResolutions {
		if r > cfg.SourceResolution {
			// we will not transcode to higher resolution than source
			continue
		}
		resMap[r] = struct{}{}
	}
	if cfg.SourceResolution == resolution.R480 {
		// normally, we won't transcode to 480p
		// however, if source video has resolution is 480, we will transcode it
		resMap[resolution.R480] = struct{}{}
	}

	res := make([]resolution.Resolution, 0, len(resMap))
	for key := range resMap {
		res = append(res, key)
	}
	// reorder res slice to make sure resolutions are in decreasing order
	// eg: [1080, 720, 360]
	sort.Slice(res, func(i, j int) bool {
		return res[i] > res[j]
	})
	return res
}

func (b *CommandBuilder) buildFilterBitRates(cfg CommandConfig) (map[resolution.Resolution]filterBitRate, []resolution.Resolution) {
	bitRates := make(map[resolution.Resolution]filterBitRate)
	res := make([]resolution.Resolution, 0, len(cfg.TargetResolutions))
	bitRate := cfg.SourceBitRate
	currentRes := cfg.TargetResolutions[0]
	for i, r := range cfg.TargetResolutions {
		bitRate = b.downBitRateValue(bitRate, currentRes, r)
		curBitRate := bitRate
		if i != 0 && currentRes != resolution.R1080 && cfg.SourceFrameRate > b.frameRateThreshold {
			// if this is not source retention and source fps > 48, we will down scale fps so scale down bitrate
			curBitRate = curBitRate * 10 / 15
		}
		if i != 0 && curBitRate < b.ignoreResolutionThreshold {
			// note that, if the bitrate of this retention is lower than ignoreResolutionThreshold, so we won't transcode it
			// except this is the source retention
			continue
		}
		if curBitRate < b.ignoreResolutionThreshold {
			curBitRate = b.ignoreResolutionThreshold
		}
		currentRes = r
		res = append(res, r)
		bitRates[r] = filterBitRate{
			inputBitRate: fmt.Sprintf("%dk", curBitRate/Kb),
			maxRate:      fmt.Sprintf("%dk", curBitRate*150/100/Kb), // max_rate = bitrate * 1.5
		}
	}
	return bitRates, res
}
