package streamer

import (
	"fmt"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

type Encoder struct {
	InputFile       string
	OutputDir       string
	SegmentDuration int
}

func New(in, out string, seg int) *Encoder {
	return &Encoder{
		InputFile:       in,
		OutputDir:       out,
		SegmentDuration: seg,
	}
}

func (e *Encoder) Encode() (string, error) {
	b := path.Base(e.InputFile)
	baseFileName := strings.TrimSuffix(b, filepath.Ext(b))

	ffmpegCmd := exec.Command(
		"ffmpeg",
		"-i", e.InputFile,
		"-map", "0:v:0",
		"-map", "0:a:0",
		"-map", "0:v:0",
		"-map", "0:a:0",
		"-map", "0:v:0",
		"-map", "0:a:0",
		"-c:v", "libx264",
		"-crf", "22",
		"-c:a", "aac",
		"-ar", "48000",
		"-filter:v:0", "scale=w=480:h=270",
		"-maxrate:v:0", "600k",
		"-b:a:0", "64k",
		"-filter:v:1", "scale=w=640:h=360",
		"-maxrate:v:1", "900k",
		"-b:a:1", "128k",
		"-filter:v:2", "scale=w=1280:h=720",
		"-maxrate:v:2", "600k",
		"-b:a:2", "64k",
		"-var_stream_map", "v:0,a:0,name:360p v:1,a:1,name:480p v:2,a:2,name:720p",
		"-preset", "slow",
		"-hls_list_size", "0",
		"-threads", "0",
		"-f", "hls",
		"-hls_playlist_type", "event",
		"-hls_time", strconv.Itoa(e.SegmentDuration),
		"-hls_flags", "independent_segments",
		"-master_pl_name", fmt.Sprintf("%s.m3u8", baseFileName),
		"-profile:v", "baseline", // baseline profile is compatible with most devices
		"-level", "3.0",
		fmt.Sprintf("%s/%s-%%v.m3u8", e.OutputDir, baseFileName),
	)

	output, err := ffmpegCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create HLS: %v\nOutput: %s", err, string(output))
	}

	return fmt.Sprintf("%s/%s.m3u8", e.OutputDir, baseFileName), nil
}
