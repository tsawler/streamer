package streamer

import (
	"fmt"
	"os/exec"
	"strconv"
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

func (e *Encoder) Encode() error {
	// Create the HLS playlist and segment the video using ffmpeg
	ffmpegCmd := exec.Command(
		"ffmpeg",
		"-i", e.InputFile,
		"-profile:v", "baseline", // baseline profile is compatible with most devices
		"-level", "3.0",
		"-start_number", "0", // start numbering segments from 0
		"-hls_time", strconv.Itoa(e.SegmentDuration), // duration of each segment in seconds
		"-hls_list_size", "0", // keep all segments in the playlist
		"-f", "hls",
		fmt.Sprintf("%s/playlist.m3u8", e.OutputDir),
	)

	output, err := ffmpegCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create HLS: %v\nOutput: %s", err, string(output))
	}

	return nil
}
