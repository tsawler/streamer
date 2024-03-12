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
		"-filter:v:0", "scale=w=1920:h=1080",
		"-maxrate:v:0", "900k",
		"-b:a:0", "64k",
		"-filter:v:1", "scale=w=1280:h=720",
		"-maxrate:v:1", "600k",
		"-b:a:1", "128k",
		"-filter:v:2", "scale=-2:480",
		"-maxrate:v:2", "400k",
		"-b:a:2", "64k",
		"-var_stream_map", "v:0,a:0,name:1080p v:1,a:1,name:720p v:2,a:2,name:480p",
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
		"-progress", "-",
		fmt.Sprintf("%s/%s-%%v.m3u8", e.OutputDir, baseFileName),
	)

	output, err := ffmpegCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create HLS: %v\nOutput: %s", err, string(output))
	}
	return fmt.Sprintf("%s/%s.m3u8", e.OutputDir, baseFileName), nil
}
