package streamer

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

type VideoProcessor struct {
	ID              int
	InputFile       string
	OutputDir       string
	SegmentDuration int
	NotifyChan      chan ProcessingMessage
}

type ProcessingMessage struct {
	ID         int
	Successful bool
	Message    string
}

func New(in, out string, seg int, c chan ProcessingMessage) *VideoProcessor {
	return &VideoProcessor{
		InputFile:       in,
		OutputDir:       out,
		SegmentDuration: seg,
		NotifyChan:      c,
	}
}

// EncodeToHLS takes input file, from receiver v.InputFile, and encodes to HLS format
// at 1080p, 720p, and 480p, putting resulting files in the output directory
// specified in the receiver as v.OutputDir.
func (v *VideoProcessor) EncodeToHLS() (*string, error) {
	// Create output directory if it does not exist.
	const mode = 0755
	if _, err := os.Stat(v.OutputDir); os.IsNotExist(err) {
		err := os.MkdirAll(v.OutputDir, mode)
		if err != nil {
			return nil, err
		}
	}
	b := path.Base(v.InputFile)
	baseFileName := strings.TrimSuffix(b, filepath.Ext(b))

	go func() {
		ffmpegCmd := exec.Command(
			"ffmpeg",
			"-i", v.InputFile,
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
			"-filter:v:0", "scale=-2:1080",
			"-maxrate:v:0", "1200k",
			"-b:a:0", "64k",
			"-filter:v:1", "scale=-2:720",
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
			"-hls_time", strconv.Itoa(v.SegmentDuration),
			"-hls_flags", "independent_segments",
			"-master_pl_name", fmt.Sprintf("%s.m3u8", baseFileName),
			"-profile:v", "baseline", // baseline profile is compatible with most devices
			"-level", "3.0",
			"-progress", "-",
			"-nostats",
			fmt.Sprintf("%s/%s-%%v.m3u8", v.OutputDir, baseFileName),
		)

		output, err := ffmpegCmd.CombinedOutput()
		successful := true
		message := "Processing complete"
		if err != nil {
			successful = false
			message = fmt.Sprintf("failed to create HLS: %v\nOutput: %s", err, string(output))
		}

		v.NotifyChan <- ProcessingMessage{
			ID:         v.ID,
			Successful: successful,
			Message:    message,
		}

	}()

	//fmt.Println(string(output))

	msg := fmt.Sprintf("%s/%s.m3u8", v.OutputDir, baseFileName)
	return &msg, nil
}

// EncodeToMP4 takes input file, from receiver v.InputFile, and encodes to MP4 format
// putting resulting file in the output directory specified in the receiver as v.OutputDir.
func (v *VideoProcessor) EncodeToMP4() (*string, error) {
	// TODO: implement logic

	return nil, nil
}

// CheckSignature returns true if the signature supplied in the URL is valid, and false
// if it is not, or does not exist.
func (v *VideoProcessor) CheckSignature(sig string) bool {
	// TODO: implement logic

	return true
}
