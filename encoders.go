package streamer

import (
	"fmt"
	"github.com/xfrr/goffmpeg/transcoder"
	"os/exec"
	"strconv"
)

// Encoder is an interface for encoding video. Any type that wants to satisfy
// this interface must implement all its methods.
type Encoder interface {
	EncodeToMP4(v *Video, baseFileName string) error
	EncodeToHLS(v *Video, baseFileName string) error
	EncodeToHLSEncrypted(v *Video, baseFileName string) error
}

// VideoEncoder is a type which satisfies the Encoder interface because it implements
// all the methods specified in Encoder.
type VideoEncoder struct{}

// EncodeToMP4 takes a Video object and a base file name, and encodes to MP4 format.
func (ve *VideoEncoder) EncodeToMP4(v *Video, baseFileName string) error {
	// Create a transcoder.
	trans := new(transcoder.Transcoder)

	// Build output path.
	outputPath := fmt.Sprintf("%s/%s.mp4", v.OutputDir, baseFileName)

	// Initialize the transcoder.
	err := trans.Initialize(v.InputFile, outputPath)
	if err != nil {
		return err
	}

	// Set codec.
	trans.MediaFile().SetVideoCodec("libx264")

	// Start transcoder process.
	done := trans.Run(false)

	// Wait for the transcoding process to end.
	err = <-done
	if err != nil {
		return err
	}

	return nil
}

// EncodeToHLS takes a Video object and a base file name, and encodes to HLS format.
func (ve *VideoEncoder) EncodeToHLS(v *Video, baseFileName string) error {
	result := make(chan error)

	go func(result chan error) {
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
			"-maxrate:v:0", v.Options.MaxRate1080p,
			"-b:a:0", "128k",
			"-filter:v:1", "scale=-2:720",
			"-maxrate:v:1", v.Options.MaxRate720p,
			"-b:a:1", "128k",
			"-filter:v:2", "scale=-2:480",
			"-maxrate:v:2", v.Options.MaxRate480p,
			"-b:a:2", "64k",
			"-var_stream_map", "v:0,a:0,name:1080p v:1,a:1,name:720p v:2,a:2,name:480p",
			"-preset", "slow",
			"-hls_list_size", "0",
			"-threads", "0",
			"-f", "hls",
			"-hls_playlist_type", "event",
			"-hls_time", strconv.Itoa(v.Options.SegmentDuration),
			"-hls_flags", "independent_segments",
			"-hls_segment_type", "mpegts",
			"-hls_playlist_type", "vod",
			"-master_pl_name", fmt.Sprintf("%s.m3u8", baseFileName),
			"-profile:v", "baseline", // baseline profile is compatible with most devices
			"-level", "3.0",
			"-progress", "-",
			"-nostats",
			fmt.Sprintf("%s/%s-%%v.m3u8", v.OutputDir, baseFileName),
		)

		_, err := ffmpegCmd.CombinedOutput()
		result <- err
	}(result)

	err := <-result
	if err != nil {
		return err
	}

	return nil
}

// EncodeToHLSEncrypted takes a Video object and a base file name, and encodes to encrypted HLS format.
func (ve *VideoEncoder) EncodeToHLSEncrypted(v *Video, baseFileName string) error {
	result := make(chan error)

	go func(result chan error) {
		ffmpegCmd := exec.Command(
			"ffmpeg",
			"-i", v.InputFile,
			"-map", "0:v:0", // We need three of this line and the next (one
			"-map", "0:a:0", // for each of the resolutions we want to encode to).
			"-map", "0:v:0",
			"-map", "0:a:0",
			"-map", "0:v:0",
			"-map", "0:a:0",
			"-c:v", "libx264", // Our video codec (H.264/MPEG-4 AVC video coding format).
			"-crf", "22",
			"-c:a", "aac",
			"-ar", "48000",
			"-filter:v:0", "scale=-2:1080", // 1080p resolution.
			"-maxrate:v:0", v.Options.MaxRate1080p,
			"-b:a:0", "128k",
			"-filter:v:1", "scale=-2:720", // 720p resolution.
			"-maxrate:v:1", v.Options.MaxRate720p,
			"-b:a:1", "128k",
			"-filter:v:2", "scale=-2:480", // 480p resolution.
			"-maxrate:v:2", v.Options.MaxRate480p,
			"-b:a:2", "64k",
			"-var_stream_map", "v:0,a:0,name:1080p v:1,a:1,name:720p v:2,a:2,name:480p", // Our map of resolutions.
			"-preset", "slow",
			"-hls_list_size", "0",
			"-threads", "0",
			"-f", "hls",
			"-hls_playlist_type", "event",
			"-hls_time", strconv.Itoa(v.Options.SegmentDuration),
			"-hls_flags", "independent_segments",
			"-hls_segment_type", "mpegts",
			"-hls_key_info_file", v.Options.KeyInfo,
			"-hls_playlist_type", "vod",
			"-master_pl_name", fmt.Sprintf("%s.m3u8", baseFileName),
			"-profile:v", "baseline", // The baseline profile is compatible with most devices.
			"-level", "3.0",
			"-progress", "-",
			"-nostats",
			fmt.Sprintf("%s/%s-%%v.m3u8", v.OutputDir, baseFileName),
		)

		_, err := ffmpegCmd.CombinedOutput()
		result <- err
	}(result)

	err := <-result
	if err != nil {
		return err
	}

	return nil
}
