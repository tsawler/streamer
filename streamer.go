package streamer

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/tsawler/signer"
	"github.com/tsawler/toolbox"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Video is the type for a video that we wish to process.
type Video struct {
	ID              int                    // An arbitrary ID for the video.
	InputFile       string                 // The path to the input file.
	OutputDir       string                 // The path to the output directory.
	NotifyChan      chan ProcessingMessage // A channel to receive the output message.
	RenameOutput    bool                   // If true, generate random name for output file.
	Secret          string                 // For encrypted HLS, the name of the file with the secret.
	KeyInfo         string                 // For encrypted HLS, the key info file.
	EncodingType    string                 // mp4, hls, or hls-encrypted.
	WebSocket       *websocket.Conn        // An (optional) websocket connection to send messages around.
	SegmentDuration int                    // If HLS, how long should segments be in seconds?
	MaxRate1080p    string                 // The Maximum rate for 1080p encoding.
	MaxRate720p     string                 // The Maximum rate for 720p encoding.
	MaxRate480p     string                 // The Maximum rate for 480p encoding.
}

// NewVideo is a convenience factory method for creating video objects with
// sensible default values.
func NewVideo(encType, max1080, max720, max480 string, rename ...bool) Video {
	if max1080 == "" {
		max1080 = "1200k"
	}
	if max720 == "" {
		max720 = "600k"
	}
	if max480 == "" {
		max480 = "400k"
	}
	if encType == "" {
		encType = "mp4"
	}
	renameFile := false
	if len(rename) > 0 {
		renameFile = rename[0]
	}
	return Video{
		EncodingType: encType,
		MaxRate1080p: max1080,
		MaxRate720p:  max720,
		MaxRate480p:  max480,
		RenameOutput: renameFile,
	}
}

// ProcessingMessage is the information sent back to the client.
type ProcessingMessage struct {
	ID         int    `json:"id"`          // The ID of the video.
	Successful bool   `json:"successful"`  // True if successfully encoded.
	Message    string `json:"message"`     // A human-readable message.
	OutputFile string `json:"output_file"` // The name of the generated file.
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
		v.sendToNotifyChan(false, "", fmt.Sprintf("error processing for %d: invalid encoding type", v.ID))
	}
}

// encodeToHLSEncrypted takes input file, from receiver v.InputFile, and encodes to HLS format
// at 1080p, 720p, and 480p, putting resulting files in the output directory
// specified in the receiver as v.OutputDir. The resulting files are encrypted.
func (v *Video) encodeToHLSEncrypted() {
	// Make sure output directory exists.
	err := v.createDirIfNotExists()
	if err != nil {
		v.sendToNotifyChan(false, "", err.Error())
		return
	}

	baseFileName := ""

	if !v.RenameOutput {
		// Get base filename.
		b := path.Base(v.InputFile)
		baseFileName = strings.TrimSuffix(b, filepath.Ext(b))
	} else {
		var t toolbox.Tools
		baseFileName = t.RandomString(10)
	}

	var ve VideoEncoder
	err = ve.EncodeToHLSEncrypted(v, baseFileName)
	if err != nil {
		v.pushToWs(fmt.Sprintf("Error processing video id %d: %s", v.ID, err.Error()))
		v.sendToNotifyChan(false, fmt.Sprintf("%s.m3u8", baseFileName), fmt.Sprintf("Error processing video id %d: %s", v.ID, err.Error()))
		return
	}

	v.pushToWs(fmt.Sprintf("Processing complete for id %d", v.ID))
	v.sendToNotifyChan(true, fmt.Sprintf("%s.m3u8", baseFileName), fmt.Sprintf("Processing complete for id %d", v.ID))
}

// encodeToHLS takes input file, from receiver v.InputFile, and encodes to HLS format
// at 1080p, 720p, and 480p, putting resulting files in the output directory
// specified in the receiver as v.OutputDir.
func (v *Video) encodeToHLS() {
	// Make sure output directory exists.
	err := v.createDirIfNotExists()
	if err != nil {
		v.sendToNotifyChan(false, "", err.Error())
		return
	}

	baseFileName := ""

	if !v.RenameOutput {
		// Get base filename.
		b := path.Base(v.InputFile)
		baseFileName = strings.TrimSuffix(b, filepath.Ext(b))
	} else {
		var t toolbox.Tools
		baseFileName = t.RandomString(10)
	}

	var ve VideoEncoder

	err = ve.EncodeToHLS(v, baseFileName)
	if err != nil {
		v.pushToWs(fmt.Sprintf("Error processing video id %d: %s", v.ID, err.Error()))
		v.sendToNotifyChan(false, fmt.Sprintf("%s.m3u8", baseFileName), fmt.Sprintf("Error processing video id %d: %s", v.ID, err.Error()))
		return
	}

	v.pushToWs(fmt.Sprintf("Processing complete for id %d", v.ID))
	v.sendToNotifyChan(true, fmt.Sprintf("%s.m3u8", baseFileName), fmt.Sprintf("Processing complete for id %d", v.ID))
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
func (v *Video) sendToNotifyChan(successful bool, fileName, message string) {
	v.NotifyChan <- ProcessingMessage{
		ID:         v.ID,
		Successful: successful,
		Message:    message,
		OutputFile: fileName,
	}
}

// encodeToMP4 takes input file, from receiver v.InputFile, and encodes to MP4 format
// putting resulting file in the output directory specified in the receiver as v.OutputDir.
func (v *Video) encodeToMP4() {
	// Make sure output directory exists.
	err := v.createDirIfNotExists()
	if err != nil {
		v.sendToNotifyChan(false, "", err.Error())
		return
	}

	baseFileName := ""

	if !v.RenameOutput {
		// Get base filename.
		b := path.Base(v.InputFile)
		baseFileName = strings.TrimSuffix(b, filepath.Ext(b))
	} else {
		var t toolbox.Tools
		baseFileName = t.RandomString(10)
	}

	var ve VideoEncoder
	err = ve.EncodeToMP4(v, baseFileName)
	if err != nil {
		v.pushToWs(err.Error())
		v.sendToNotifyChan(false, "", err.Error())
		return
	}

	v.pushToWs(fmt.Sprintf("Encoding successful for id %d", v.ID))
	v.sendToNotifyChan(true, fmt.Sprintf("%s.mp4", baseFileName), fmt.Sprintf("Encoding successful for id %d", v.ID))
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
