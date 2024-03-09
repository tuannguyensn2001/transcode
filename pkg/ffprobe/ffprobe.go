package ffprobe

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"os/exec"
	"strconv"
	"strings"
	"transcode/pkg/config"
	"transcode/pkg/resolution"

	"github.com/thnthien/great-deku/container"
	"github.com/thnthien/great-deku/l"
)

type Ffprobe struct {
	ll l.Logger `container:"name"`

	ffmpegBin  string
	ffprobeBin string
}

type InputInfo struct {
	Width        int64                 `json:"width"`
	Height       resolution.Resolution `json:"height"`
	Duration     int                   `json:"duration"`
	BitRate      int64                 `json:"bit_rate"`
	AudioBitRate int64                 `json:"audio_bit_rate"`
	FrameRate    int                   `json:"frame_rate"` //fps
}

func (i *InputInfo) setValue(args []string) {
	if len(args) < 2 {
		return
	}

	switch args[0] {
	case "width":
		i.Width, _ = strconv.ParseInt(args[1], 10, 64)
	case "height":
		height, _ := strconv.ParseInt(args[1], 10, 64)
		i.Height = resolution.Resolution(height)
	case "duration":
		fDur, _ := strconv.ParseFloat(args[1], 64)
		i.Duration = int(math.Round(fDur))
	case "bit_rate":
		i.BitRate, _ = strconv.ParseInt(args[1], 10, 64)
	case "r_frame_rate":
		rs := strings.Split(args[1], "/")
		frameRate, _ := strconv.ParseInt(rs[0], 10, 64)
		sec, _ := strconv.ParseFloat(rs[1], 64)
		if sec != 0 {
			frameRate = int64(math.Round(float64(frameRate) / sec))
		}
		i.FrameRate = int(frameRate)
	}
}

func New(cfg config.ServerConfig) *Ffprobe {
	f := &Ffprobe{
		ffmpegBin:  cfg.FfmpegBin,
		ffprobeBin: cfg.FfprobeBin,
	}
	container.Fill(f)
	return f
}

func (f *Ffprobe) exec(cmd *exec.Cmd) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	out, err := stdout.String(), errors.New(stderr.String())
	if err.Error() != "" {
		return out, err
	}
	return out, nil
}

func (f *Ffprobe) FileDuration(filePath string) (string, error) {
	cmd := exec.Command(f.ffprobeBin, []string{
		"-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", filePath,
	}...)

	out, err := f.exec(cmd)

	if err != nil && strings.Contains(err.Error(), "non-existing SPS 0 referenced in buffering period") {
		f.ll.Error("actual error will be hidden", l.Error(err))
		err = nil
	}

	if len(out) > 0 {
		out = out[:len(out)-1]
	}

	return out, err
}

// InputInfo
// input: maybe the filepath or can be the rtmp url
// readIntervals: how many secs should read to know the info of input
func (f *Ffprobe) InputInfo(input string, readIntervals int) (*InputInfo, error) {
	// ffprobe -v error -read_intervals "%+2" -select_streams v:0
	// -show_entries stream=width,height,duration,bit_rate,r_frame_rate -of default=noprint_wrappers=1 rtmp://127.0.0.1:1935/live/7868802855338312

	//region read video info
	cmd := exec.Command(f.ffprobeBin, []string{
		"-v", "error", "-read_intervals", fmt.Sprintf("%%+%d", readIntervals), "-select_streams", "v:0",
		"-show_entries", "stream=width,height,duration,bit_rate,r_frame_rate", "-of", "default=noprint_wrappers=1", input,
	}...)
	out, err := f.exec(cmd)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(out, "\n")
	info := &InputInfo{}

	for _, line := range lines {
		info.setValue(parseValue(line))
	}
	//endregion

	//region read audio bitrate
	cmd = exec.Command(f.ffprobeBin, []string{
		"-v", "error", "-read_intervals", fmt.Sprintf("%%+%d", readIntervals), "-select_streams", "a:0",
		"-show_entries", "stream=bit_rate", "-of", "default=noprint_wrappers=1", input,
	}...)
	out, err = f.exec(cmd)
	if err != nil {
		return nil, err
	}
	lines = strings.Split(out, "\n")
	info.AudioBitRate, _ = strconv.ParseInt(getValue(lines[0]), 10, 64)
	//endregion

	return info, nil
}

func parseValue(input string) []string {
	return strings.Split(input, "=")
}

func getValue(input string) string {
	strs := strings.Split(input, "=")
	if len(strs) < 2 {
		return input
	}
	return strs[1]
}
