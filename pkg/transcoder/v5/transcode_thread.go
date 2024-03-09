package v5

import (
	"regexp"
	"strings"
	"sync"
	ffmpegrunner "transcode/pkg/ffmpeg_runner"
	"transcode/pkg/ffprobe"
	"transcode/pkg/transcoder"

	"github.com/thnthien/great-deku/container"
	"github.com/thnthien/great-deku/l"
	"github.com/thnthien/great-deku/rpooling"
)

var (
	m3u8Regex     = regexp.MustCompile(`.?(stream_\d\.m3u8).?`)
	dataNameRegex = regexp.MustCompile(`data(\d.+?)\.`)
)

type transcodeThread struct {
	ll      l.Logger         `container:"name"`
	ffprobe *ffprobe.Ffprobe `container:"name"`

	pool       rpooling.IPool
	wg         *sync.WaitGroup
	clearData  bool
	outputPath string
	baseKey    string
	messages   chan ffmpegrunner.OpeningFileProgress
	lastTSFile transcoder.UploadFile
	outputChan chan transcoder.UploadFile
}

func newThread(outputPath, baseKey string, clearData bool, outputChan chan transcoder.UploadFile, wg *sync.WaitGroup) *transcodeThread {
	t := &transcodeThread{
		wg:         wg,
		clearData:  clearData,
		outputPath: outputPath,
		baseKey:    baseKey,
		messages:   make(chan ffmpegrunner.OpeningFileProgress, 100),
		outputChan: outputChan,
	}
	container.Fill(t)
	t.pool = rpooling.New(10, t.ll)

	return t
}

// run starting to handle uploading files for a resolution
func (t *transcodeThread) run() {
	go t.processMessage() // process logs of ffmpeg
}

// stop shutdown thread
func (t *transcodeThread) stop() {
	close(t.messages)
}

// processMessage process logs of ffmpeg
func (t *transcodeThread) processMessage() {
	wg := sync.WaitGroup{}
	for m := range t.messages {
		isM3U8 := m3u8Regex.MatchString(m.FilePath)
		if isM3U8 {
			t.pool.Submit(func() {
				elements := strings.Split(m.FilePath, "/")
				t.uploadFile(transcoder.UploadFile{
					Name: elements[len(elements)-1],
					Path: m.FilePath,
				}, &wg)
			})
			continue
		}

		if t.lastTSFile.Name != "" {
			// this is not the first time, upload last ts file and update m3u8 file
			t.uploadFile(t.lastTSFile, &wg)
		}

		//region update lastTsFile
		t.lastTSFile.Path = m.FilePath
		if strings.HasSuffix(t.lastTSFile.Path, ".tmp") {
			t.lastTSFile.Path = t.lastTSFile.Path[:len(t.lastTSFile.Path)-4]
		}
		elements := strings.Split(t.lastTSFile.Path, "/")
		t.lastTSFile.Name = elements[len(elements)-1]
		//endregion
	}

	// after call stop thread and done process all messages
	// handle the last segment file
	if t.lastTSFile.Name != "" {
		t.uploadFile(t.lastTSFile, &wg)
	}
	wg.Wait()
	t.wg.Done()
}

// uploadFile this function is used to upload file
func (t *transcodeThread) uploadFile(file transcoder.UploadFile, wg *sync.WaitGroup) {
	wg.Add(1)
	t.pool.Submit(func() {
		defer wg.Done()
		file.UploadKey = t.baseKey + "/" + file.Name
		t.outputChan <- file
	})
}
