package streamer

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
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
	WebSocket       *websocket.Conn        // An (optional) websocket connection to send messages around.
}

// ProcessingMessage is the information sent back to the client.
type ProcessingMessage struct {
	ID         int    `json:"id"`         // The ID of the video.
	Successful bool   `json:"successful"` // True if successfully encoded.
	Message    string `json:"message"`    // A human-readable message.
}

// ToJSON marshals the receiver, pm, to JSON and returns a slice of bytes.
func (pm *ProcessingMessage) ToJSON() ([]byte, error) {
	b, err := json.Marshal(pm)
	if err != nil {
		return nil, err
	}

	return b, nil
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
func (v *Video) encode() {
	v.pushToWs(fmt.Sprintf("Processing started for %d", v.ID))

	switch v.EncodingType {
	case "mp4":
		v.encodeToMP4()
	case "hls":
		v.encodeToHLS()
	case "hls-encrypted":
		v.encodeToHLSEncrypted()
	default:
		v.pushToWs(fmt.Sprintf("error processing for %d: invalid encoding type", v.ID))
		v.sendToNotifyChan(false, fmt.Sprintf("error processing for %d: invalid encoding type", v.ID))
	}
}

// encodeToHLSEncrypted takes input file, from receiver v.InputFile, and encodes to HLS format
// at 1080p, 720p, and 480p, putting resulting files in the output directory
// specified in the receiver as v.OutputDir. The resulting files are encrypted.
func (v *Video) encodeToHLSEncrypted() {
	// Make sure output directory exists.
	err := v.createDirIfNotExists()
	if err != nil {
		v.sendToNotifyChan(false, err.Error())
		return
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

		_, err = ffmpegCmd.CombinedOutput()
		if err != nil {
			v.pushToWs(fmt.Sprintf("Processing failed for id %d: %s", v.ID, err.Error()))
			v.sendToNotifyChan(false, err.Error())
			return
		}

		v.pushToWs(fmt.Sprintf("Processing complete for id %d", v.ID))
		v.sendToNotifyChan(true, fmt.Sprintf("Processing complete for id %d", v.ID))
	}()
}

// encodeToHLS takes input file, from receiver v.InputFile, and encodes to HLS format
// at 1080p, 720p, and 480p, putting resulting files in the output directory
// specified in the receiver as v.OutputDir.
func (v *Video) encodeToHLS() {
	// Make sure output directory exists.
	err := v.createDirIfNotExists()
	if err != nil {
		v.sendToNotifyChan(false, err.Error())
		return
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

		_, err = ffmpegCmd.CombinedOutput()
		if err != nil {
			v.pushToWs(fmt.Sprintf("Processing failed for id %d: %s", v.ID, err.Error()))
			v.sendToNotifyChan(false, err.Error())
			return
		}

		v.pushToWs(fmt.Sprintf("Processing complete for id %d", v.ID))
		v.sendToNotifyChan(true, fmt.Sprintf("Processing complete for id %d", v.ID))
	}()
}

// createDirIfNotExists creates the output directory, and all required
// parent directories, if it does not already exist.
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

// pushToWs pushes a message to websocket, if appropriate.
func (v *Video) pushToWs(msg string) {
	if v.WebSocket != nil {
		_ = v.WebSocket.WriteMessage(websocket.TextMessage, []byte(msg))
	}
}

// pushJSONToWs pushes a message to websocket, if appropriate.
func (v *Video) pushJSONToWs(payload map[string]string) {
	if v.WebSocket != nil {
		p, err := json.Marshal(payload)
		if err == nil {
			_ = v.WebSocket.WriteJSON(v.WebSocket.WriteJSON(p))
		}
	}
}

// sendToNotifyChan pushes a message down the notify channel.
func (v *Video) sendToNotifyChan(successful bool, message string) {
	v.NotifyChan <- ProcessingMessage{
		ID:         v.ID,
		Successful: successful,
		Message:    message,
	}
}

// encodeToMP4 takes input file, from receiver v.InputFile, and encodes to MP4 format
// putting resulting file in the output directory specified in the receiver as v.OutputDir.
func (v *Video) encodeToMP4() {
	// Make sure output directory exists.
	err := v.createDirIfNotExists()
	if err != nil {
		v.sendToNotifyChan(false, err.Error())
		return
	}

	trans := new(transcoder.Transcoder)
	b := path.Base(v.InputFile)
	baseFileName := strings.TrimSuffix(b, filepath.Ext(b))
	outputPath := fmt.Sprintf("%s/%s.mp4", v.OutputDir, baseFileName)

	err = trans.Initialize(v.InputFile, outputPath)
	if err != nil {
		v.sendToNotifyChan(false, err.Error())
		return
	}

	// set codec
	trans.MediaFile().SetVideoCodec("libx264")

	// Start transcoder process with progress checking
	done := trans.Run(true)

	go func() {
		// Returns a channel to get the transcoding progress
		progress := trans.Output()

		// Printing transcoding progress to log
		curProgress := 0
		oldInt := 0

		for msg := range progress {
			if int(msg.Progress)%2 == 0 {
				if oldInt != int(msg.Progress) {
					// we have moved up 2%
					curProgress = curProgress + 2
					oldInt = int(msg.Progress)
					log.Printf("%d: %d%%\n", v.ID, curProgress)

					data := map[string]string{
						"message":  "progress",
						"video_id": fmt.Sprintf("%d", v.ID),
						"percent":  fmt.Sprintf("%d", int(msg.Progress)),
					}
					v.pushJSONToWs(data)
					v.pushToWs(fmt.Sprintf("%d", int(msg.Progress)))
				}
			}
		}
	}()

	// This channel is used to wait for the transcoding process to end
	result := <-done
	if result != nil {
		v.pushToWs(result.Error())
		v.sendToNotifyChan(false, result.Error())
		return
	}
	v.pushToWs(fmt.Sprintf("Encoding successful for id %d", v.ID))
	v.sendToNotifyChan(true, fmt.Sprintf("Encoding successful for id %d", v.ID))
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
