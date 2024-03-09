package v5

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sync"
	"time"
	"transcode/pkg/config"
	"transcode/pkg/datetime"
	ffmpegrunner "transcode/pkg/ffmpeg_runner"
	"transcode/pkg/ffprobe"
	"transcode/pkg/request"
	"transcode/pkg/resolution"
	"transcode/pkg/transcoder"

	"github.com/thnthien/great-deku/container"
	"github.com/thnthien/great-deku/l"
)

var (
	streamRegex = regexp.MustCompile(`.?(stream_\d).?`)
	masterRegex = regexp.MustCompile(`.?(master\.m3u8).?`)
	dataRegex   = regexp.MustCompile(`.?(data(\d+?)\.ts).?`)
)

type transcoderImpl struct {
	ll             l.Logger         `container:"name"`
	ffprobe        *ffprobe.Ffprobe `container:"name"`
	commandBuilder *CommandBuilder  `container:"name"`

	wg           *sync.WaitGroup
	cfg          config.ServerConfig
	runner       *ffmpegrunner.FfmpegRunner
	req          request.TranscodeReq
	threads      map[string]*transcodeThread
	uploadMaster chan struct{}
	resolutions  []resolution.Resolution
	outputChan   chan transcoder.UploadFile

	err error
}

func New(cfg config.ServerConfig, req request.TranscodeReq) *transcoderImpl {
	t := &transcoderImpl{
		wg:           &sync.WaitGroup{},
		cfg:          cfg,
		req:          req,
		runner:       ffmpegrunner.New(cfg.FfmpegBin, cfg.FfprobeBin),
		threads:      make(map[string]*transcodeThread),
		uploadMaster: make(chan struct{}),
		outputChan:   make(chan transcoder.UploadFile, 10),
	}
	os.MkdirAll(req.StoredFolderPath, 0755) //create folder for storing files
	container.Fill(t)

	return t
}

// Transcode start to transcode stream
// the flow is:
// - get the information of input stream for knows the input bit rate of streamer
// - using the information above to build command with setting match specified with the request and the input information
// - start to transcode
func (t *transcoderImpl) Transcode(ctx context.Context) (transcoder.OutputData, error) {
	defer close(t.outputChan)
	data := transcoder.OutputData{}
	//region get input stream information
	info, err := t.ffprobe.InputInfo(t.req.FilePath, 2)
	if err != nil {
		t.ll.Error("cannot get file info", l.Error(err))
		return data, err
	}
	t.ll.Info("got input info", l.Object("info", info))
	//endregion
	data.Width = int(info.Width)
	data.Resolution = int(info.Height)
	data.FPS = info.FrameRate
	data.Duration = info.Duration
	data.VideoBitrate = int(info.BitRate)
	data.AudioBitrate = int(info.AudioBitRate)

	//get the command
	args, resolutions := t.commandBuilder.buildCommand(CommandConfig{
		FolderName:         t.req.FolderName,
		FilePath:           t.req.FilePath,
		StoredFolderPath:   t.req.StoredFolderPath,
		KeyInfoFilePath:    t.req.KeyInfoFilePath,
		TargetResolutions:  t.req.Resolutions,
		SourceResolution:   info.Height,
		SourceWidth:        info.Width,
		SourceHeight:       int64(info.Height),
		SourceDuration:     info.Duration,
		SourceBitRate:      info.BitRate,
		SourceAudioBitRate: info.AudioBitRate,
		SourceFrameRate:    info.FrameRate,
	})
	if len(resolutions) == 0 {
		return transcoder.OutputData{}, errors.New("original resolution is too low")
	}
	numberOfThread := len(resolutions)
	t.resolutions = resolutions
	data.Resolutions = resolutions

	t.ll.Info("start transcode file", l.String("input", t.req.FilePath))
	t.ll.Info("ffmpeg command", l.String("command", fmt.Sprintf("%v", args)))

	for i := 0; i < numberOfThread; i++ {
		// base on the required resolutions that request want
		// so each resolution will be handled by a thread for uploading ts files, updating realtime m3u8 files
		m3u8Name := fmt.Sprintf("stream_%d.m3u8", i)
		t.wg.Add(1)
		th := newThread(t.cfg.OutputPath, t.req.FolderName, t.cfg.ClearAfterStream, t.outputChan, t.wg)
		t.threads[fmt.Sprintf("stream_%d", i)] = th
		th.run()
		t.ll.Info("start thread", l.Int64("resolution", int64(resolutions[i])),
			l.String("m3u8_name", m3u8Name), l.Int64("next_segment", 0))
	}

	startTime := datetime.Now()
	t.runner.SetArgs(args)
	done := t.runner.Run()
	logs := t.runner.Logs()

	t.handleProcess(done, logs) // handles logs of ffmpeg and controls uploading threads
	err = t.Stop(false)
	stopTime := datetime.Now()
	data.TranscodeDuration = int(startTime.DiffAbsInSeconds(stopTime))
	if t.err != nil {
		err = t.err
	}
	return data, err
}

// Stop if we want to stop or pause transcoding of stream, call to this thread
// isPause: is pausing or stopping transcoding
func (t *transcoderImpl) Stop(isPause bool) error {
	if t.runner.IsRunning() {
		if err := t.runner.Stop(); err != nil {
			// stop ffmpeg command
			return err
		}
	}

	return nil
}

func (t *transcoderImpl) Output() chan transcoder.UploadFile {
	return t.outputChan
}

// handleProcess read logs of ffmpeg and controls uploading threads
// done: channel for done signal
// logs: channel for receiving logs of ffmpeg
func (t *transcoderImpl) handleProcess(done chan error, logs chan ffmpegrunner.IProgress) {
	for {
		select {
		case err := <-done:
			// received done signal
			if err != nil {
				t.err = err
				t.ll.Error("error when handle process", l.Error(err))
			}

			for key := range t.threads {
				// call stop to all uploading threads
				t.threads[key].stop()
			}
			t.wg.Wait()
			t.ll.Info("finished transcode file", l.Object("request", t.req))

			return
		case msg := <-logs:
			if msg == nil {
				continue
			}
			switch msg.GetType() {
			case ffmpegrunner.OpeningFile:
				// if this is the opening file log, we send it to uploading thread that in charging of this file
				p := msg.(*ffmpegrunner.OpeningFileProgress)
				t.ll.Trace("opening file message", l.String("file_path", p.FilePath))
				t.handleOutputFile(p)
			default:
				t.ll.Trace("raw message", l.String("msg", msg.ToString()))
			}
		}
	}
}

func (t *transcoderImpl) handleOutputFile(p *ffmpegrunner.OpeningFileProgress) {
	filePath := p.FilePath

	if master := masterRegex.FindStringSubmatch(filePath); len(master) > 1 {
		// if this is master file, upload it immediately
		t.uploadMasterFile()
		return
	}

	//this is the case of stream file
	match := streamRegex.FindStringSubmatch(filePath)
	if len(match) < 2 {
		t.ll.Error("cannot find stream from Path", l.String("file_path", filePath))
		return
	}
	streamName := match[1]

	// we get the in charged thread and send the log to that thread
	th, ok := t.threads[streamName]
	if !ok {
		t.ll.Error("cannot find thread of stream", l.String("stream_name", streamName))
	} else {
		th.messages <- *p
	}
}

// uploadMasterFile upload master file to storage
func (t *transcoderImpl) uploadMasterFile() {
	time.Sleep(500 * time.Millisecond)
	fileName := "master.m3u8"
	filePath := filepath.Join(t.req.StoredFolderPath, fileName)
	//content, err := os.ReadFile(filePath)
	//for i := 0; i < len(t.resolutions); i++ {
	//	streamName := fmt.Sprintf("stream_%d", i)
	//	folderName := fmt.Sprintf("%d", t.resolutions[i])
	//	content = bytes.Replace(content, []byte(streamName), []byte(path.Join(folderName, streamName)), -1)
	//}
	//if err = os.WriteFile(filePath, content, 0666); err != nil {
	//	t.ll.Error("cannot update master file", l.String("file_path", filePath), l.Error(err))
	//}
	t.outputChan <- transcoder.UploadFile{
		Name:      fileName,
		Path:      filePath,
		UploadKey: path.Join(t.req.FolderName, fileName),
	}
}

// clearStream clear all files of this streaming session
func (t *transcoderImpl) clearStream() {
	err := os.RemoveAll(t.req.StoredFolderPath)
	if err != nil {
		t.ll.Error("cannot clear stream data", l.String("folder", t.req.StoredFolderPath), l.Error(err))
	} else {
		t.ll.Info("cleared stream data", l.String("folder", t.req.StoredFolderPath))
	}
}
