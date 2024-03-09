package ffprobe

import (
	"strconv"
	"strings"
	"transcode/pkg/commander"
)

func (f *Ffprobe) ReadPacket(input string) *ReadPacketor {
	args := []string{
		"-show_packets", "-show_entries",
		"packet=codec_type,duration_time,size,flags", input,
	}
	return &ReadPacketor{
		Commander: commander.New(f.ffprobeBin, args...),
	}
}

type PacketMediaType string

const (
	VideoPacket PacketMediaType = "video"
	AudioPacket PacketMediaType = "audio"
)

type Packet struct {
	MediaType    PacketMediaType `json:"media_type"`
	KeyFrame     int             `json:"key_frame"`
	DurationTime float64         `json:"duration_time"`
	Size         int64           `json:"size"`
}

type ReadPacketor struct {
	commander.Commander
}

func (r *ReadPacketor) Logs() chan Packet {
	packets := make(chan Packet)
	go r.handleLogs(packets)

	return packets
}

func (r *ReadPacketor) handleLogs(packets chan Packet) {
	defer close(packets)
	ls := r.Commander.StdoutLogs()

	var f Packet
	for line := range ls {
		if line == "[PACKET]" {
			f.DurationTime = 0
			f.MediaType = ""
			f.KeyFrame = 0
			f.Size = 0
		} else if line == "[/PACKET]" {
			packets <- f
		} else {
			value := getValue(line)
			if strings.HasPrefix(line, "codec_type") {
				f.MediaType = PacketMediaType(value)
			} else if strings.HasPrefix(line, "flags") {
				if strings.HasPrefix(value, "K") {
					f.KeyFrame = 1
				} else {
					f.KeyFrame = 0
				}
			} else if strings.HasPrefix(line, "duration_time") {
				val, _ := strconv.ParseFloat(value, 64)
				f.DurationTime = val
			} else if strings.HasPrefix(line, "size") {
				val, _ := strconv.ParseInt(value, 10, 64)
				f.Size = val
			}
		}
	}
}
