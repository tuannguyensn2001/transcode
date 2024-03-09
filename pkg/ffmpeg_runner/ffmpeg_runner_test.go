package ffmpegrunner

//func TestRunner(t *testing.T) {
//	r := New("/usr/bin/ffmpeg", "/usr/bin/ffprobe")
//	args := []string{
//		"-i", "rtmp://localhost:1935/live/1205232902078740", "-filter_complex",
//		"[0:v]split=2[v1][v2];[v1]copy[v1out];[v2]scale=w=1280:h=720[v2out]",
//		"-map", "[v1out]", "-c:v:0", "libx264", "-forced-idr", "1", "-b:v:0", "5M",
//		"-maxrate:v:0", "5M", "-minrate:v:0", "5M", "-bufsize:v:0", "10M", "-preset", "medium",
//		"-sc_threshold", "0", "-force_key_frames", "expr:gte(t,n_forced*2)",
//		"-map", "[v2out]", "-c:v:1", "libx264", "-forced-idr", "1", "-b:v:1", "3M",
//		"-maxrate:v:1", "3M", "-minrate:v:1", "3M", "-bufsize:v:1", "3M", "-preset", "medium",
//		"-sc_threshold", "0", "-force_key_frames", "expr:gte(t,n_forced*2)",
//		"-map", "a:0", "-c:a:0", "aac", "-b:a:0", "96k", "-ac", "2",
//		"-map", "a:0", "-c:a:1", "aac", "-b:a:1", "96k", "-ac", "2",
//		"-f", "hls",
//		"-hls_time", "4", "-hls_list_size", "3", "-hls_flags", "independent_segments",
//		"-hls_segment_type", "mpegts", "-hls_segment_filename", "stream_%v/data%02d.ts",
//		"-master_pl_name", "master.m3u8", "-var_stream_map", "v:0,a:0 v:1,a:1", "stream_%v.m3u8",
//	}
//	r.SetRawArgs(args)
//
//	done := r.Run()
//	logs := r.Logs()
//
//	for {
//		select {
//		case err := <-done:
//			log.Printf("error: %s", err)
//			assert.NoError(t, err)
//		case msg := <-logs:
//			if msg == nil {
//				return
//			}
//			log.Printf("%s %s", msg.GetType(), msg.ToString())
//		}
//	}
//}
