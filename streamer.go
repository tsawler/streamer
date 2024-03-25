package streamer

import (
	"errors"
	"fmt"
	"github.com/tsawler/signer"
	"github.com/xfrr/goffmpeg/transcoder"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

// Video is the type for videos we want to work with.
type Video struct {
	ID              int                    // An arbitrary ID for the video.
	InputFile       string                 // The path to the input file.
	OutputDir       string                 // The path to the output directory.
	SegmentDuration int                    // If HLS, how long should segments be in seconds?
	NotifyChan      chan ProcessingMessage // A channel to receive the output message.
	Secret          string                 // For encrypted HLS, the name of the file with the secret.
	KeyInfo         string                 // For encrypted HLS, the key info file.
	EncodingType    string                 // mp4, hls, or hls-encrypted.
}

// ProcessingMessage is the information sent back to the client.
type ProcessingMessage struct {
	ID         int    // The ID of the video.
	Successful bool   // True if successfully encoded.
	Message    string // A human-readable message.
}

// New creates and returns a new worker pool.
func New(jobQueue chan VideoProcessingJob, maxWorkers int) *VideoDispatcher {
	workerPool := make(chan chan VideoProcessingJob, maxWorkers)

	return &VideoDispatcher{
		jobQueue:   jobQueue,
		maxWorkers: maxWorkers,
		WorkerPool: workerPool,
	}
}

// encode allows us to encode the source file to one of the supported formats.
func (v *Video) encode() (*string, error) {
	switch v.EncodingType {
	case "mp4":
		return v.encodeToMP4()
	case "hls":
		return v.encodeToHLS()
	case "hls-encrypted":
		return v.encodeToHLSEncrypted()
	default:
		return nil, errors.New("invalid encoding type")
	}
}

// encodeToHLSEncrypted takes input file, from receiver v.InputFile, and encodes to HLS format
// at 1080p, 720p, and 480p, putting resulting files in the output directory
// specified in the receiver as v.OutputDir. The resulting files are encrypted.
func (v *Video) encodeToHLSEncrypted() (*string, error) {
	// Make sure output directory exists.
	err := v.createDirIfNotExists()
	if err != nil {
		return nil, err
	}

	// Get base filename.
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
			"-hls_segment_type", "mpegts",
			"-hls_key_info_file", v.KeyInfo,
			"-hls_playlist_type", "vod",
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

	msg := fmt.Sprintf("%s/%s.m3u8", v.OutputDir, baseFileName)
	return &msg, nil
}

// encodeToHLS takes input file, from receiver v.InputFile, and encodes to HLS format
// at 1080p, 720p, and 480p, putting resulting files in the output directory
// specified in the receiver as v.OutputDir.
func (v *Video) encodeToHLS() (*string, error) {
	// Make sure output directory exists.
	err := v.createDirIfNotExists()
	if err != nil {
		return nil, err
	}

	// Get base filename.
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
			"-hls_segment_type", "mpegts",
			"-hls_playlist_type", "vod",
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

	msg := fmt.Sprintf("%s/%s.m3u8", v.OutputDir, baseFileName)
	return &msg, nil
}

func (v *Video) createDirIfNotExists() error {
	// Create output directory if it does not exist.
	const mode = 0755
	if _, err := os.Stat(v.OutputDir); os.IsNotExist(err) {
		err := os.MkdirAll(v.OutputDir, mode)
		if err != nil {
			return err
		}
	}
	return nil
}

// encodeToMP4 takes input file, from receiver v.InputFile, and encodes to MP4 format
// putting resulting file in the output directory specified in the receiver as v.OutputDir.
func (v *Video) encodeToMP4() (*string, error) {
	successful := true
	message := "Processing complete"

	// Make sure output directory exists.
	err := v.createDirIfNotExists()
	if err != nil {
		return nil, err
	}

	trans := new(transcoder.Transcoder)
	b := path.Base(v.InputFile)
	baseFileName := strings.TrimSuffix(b, filepath.Ext(b))
	outputPath := fmt.Sprintf("%s/%s.mp4", v.OutputDir, baseFileName)
	go func() {
		err = trans.Initialize(v.InputFile, outputPath)
		if err != nil {
			log.Println(err)
			successful = false
			message = "Failed to initialize"
		}

		// set codec
		trans.MediaFile().SetVideoCodec("libx264")

		// Start transcoder process with progress checking
		done := trans.Run(true)
		err = <-done
		if err != nil {
			successful = false
			message = fmt.Sprintf("failed to create MP$: %v\n", err)
		}

		v.sendToResultChan(successful, message)
	}()

	return &outputPath, nil
}

func (v *Video) sendToResultChan(successful bool, message string) {
	v.NotifyChan <- ProcessingMessage{
		ID:         v.ID,
		Successful: successful,
		Message:    message,
	}
}

// CheckSignature returns true if the signature supplied in the URL is valid, and false
// if it is not, or does not exist. It also returns false if the expiration time (minutes)
// has passed.
func (v *Video) CheckSignature(urlPath string, expiration int) bool {
	sign := signer.Signature{Secret: v.Secret}
	valid, err := sign.VerifyURL(urlPath)
	if err != nil {
		return false
	}

	valid = sign.Expired(urlPath, expiration)
	return valid
}
