package streamer

import (
	"fmt"
	"github.com/tsawler/toolbox"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Processor is a simple wrapper for anything that satisfies the Encoder interface.
// This makes testing possible without actually running ffmpeg.
type Processor struct {
	Engine Encoder
}

// ProcessingMessage is the information sent back to the client.
type ProcessingMessage struct {
	ID         int    `json:"id"`          // The ID of the video.
	Successful bool   `json:"successful"`  // True if successfully encoded.
	Message    string `json:"message"`     // A human-readable message.
	OutputFile string `json:"output_file"` // The name of the generated file.
}

// Video is the type for a video that we wish to process.
type Video struct {
	ID           int                    // An arbitrary ID for the video.
	InputFile    string                 // The path to the input file.
	OutputDir    string                 // The path to the output directory.
	EncodingType string                 // mp4, hls, or hls-encrypted.
	NotifyChan   chan ProcessingMessage // A channel to receive the output message.
	Options      *VideoOptions          // Options for encoding.
	Encoder      Processor              // The processing engine we'll use for encoding.
}

// New creates and returns a new worker pool.
func New(jobQueue chan VideoProcessingJob, maxWorkers int, encoder ...Processor) *VideoDispatcher {
	workerPool := make(chan chan VideoProcessingJob, maxWorkers)
	var p Processor
	if len(encoder) > 0 {
		p = Processor{
			Engine: encoder[0].Engine,
		}
	} else {
		var e VideoEncoder
		p = Processor{
			Engine: &e,
		}
	}
	return &VideoDispatcher{
		jobQueue:   jobQueue,
		maxWorkers: maxWorkers,
		WorkerPool: workerPool,
		Processor:  p,
	}
}

type VideoOptions struct {
	RenameOutput    bool   // If true, generate random name for output file.
	Secret          string // For encrypted HLS, the name of the file with the secret.
	KeyInfo         string // For encrypted HLS, the key info file.
	SegmentDuration int    // If HLS, how long should segments be in seconds?
	MaxRate1080p    string // The Maximum rate for 1080p encoding.
	MaxRate720p     string // The Maximum rate for 720p encoding.
	MaxRate480p     string // The Maximum rate for 480p encoding.
}

// NewVideo is a convenience factory method for creating video objects with
// sensible default values.
func (vd *VideoDispatcher) NewVideo(id int, input, output, encType string, notifyChan chan ProcessingMessage, ops *VideoOptions) Video {
	if ops == nil {
		ops = &VideoOptions{}
	}
	if ops.MaxRate1080p == "" {
		ops.MaxRate1080p = "1200k"
	}
	if ops.MaxRate720p == "" {
		ops.MaxRate720p = "600k"
	}
	if ops.MaxRate480p == "" {
		ops.MaxRate480p = "400k"
	}
	if encType == "" {
		encType = "mp4"
	}
	return Video{
		ID:           id,
		InputFile:    input,
		OutputDir:    output,
		EncodingType: encType,
		NotifyChan:   notifyChan,
		Encoder:      vd.Processor,
		Options:      ops,
	}
}

// encode allows us to encode the source file to one of the supported formats.
func (v *Video) encode() {
	var fileName string

	switch v.EncodingType {
	case "mp4":
		name, err := v.encodeToMP4()
		if err != nil {
			v.sendToNotifyChan(false, "", fmt.Sprintf("error processing %d: %s", v.ID, err.Error()))
			return
		}
		fileName = fmt.Sprintf("%s.mp4", name)
	case "hls":
		name, err := v.encodeToHLS()
		if err != nil {
			v.sendToNotifyChan(false, "", fmt.Sprintf("error processing %d: %s", v.ID, err.Error()))
			return
		}
		fileName = fmt.Sprintf("%s.m3u8", name)
	case "hls-encrypted":
		name, err := v.encodeToHLSEncrypted()
		if err != nil {
			v.sendToNotifyChan(false, "", fmt.Sprintf("error processing %d: %s", v.ID, err.Error()))
			return
		}
		fileName = fmt.Sprintf("%s.m3u8", name)
	default:
		v.sendToNotifyChan(false, "", fmt.Sprintf("error processing for %d: invalid encoding type", v.ID))
		return
	}

	// Encoding was successful.
	v.sendToNotifyChan(true, fileName, fmt.Sprintf("Video ID #%d processed and saved as %s", v.ID, fmt.Sprintf("%s/%s", v.OutputDir, fileName)))
}

// encodeToHLSEncrypted takes input file, from receiver v.InputFile, and encodes to HLS format
// at 1080p, 720p, and 480p, putting resulting files in the output directory
// specified in the receiver as v.OutputDir. The resulting files are encrypted.
func (v *Video) encodeToHLSEncrypted() (string, error) {
	// Make sure output directory exists.
	err := v.createDirIfNotExists()
	if err != nil {
		return "", err
	}

	baseFileName := ""

	if !v.Options.RenameOutput {
		// Get base filename.
		b := path.Base(v.InputFile)
		baseFileName = strings.TrimSuffix(b, filepath.Ext(b))
	} else {
		var t toolbox.Tools
		baseFileName = t.RandomString(10)
	}

	err = v.Encoder.Engine.EncodeToHLSEncrypted(v, baseFileName)
	if err != nil {
		return "", err
	}

	return baseFileName, nil
}

// encodeToHLS takes input file, from receiver v.InputFile, and encodes to HLS format
// at 1080p, 720p, and 480p, putting resulting files in the output directory
// specified in the receiver as v.OutputDir.
func (v *Video) encodeToHLS() (string, error) {
	// Make sure output directory exists.
	err := v.createDirIfNotExists()
	if err != nil {
		return "", err
	}

	baseFileName := ""

	if !v.Options.RenameOutput {
		// Get base filename.
		b := path.Base(v.InputFile)
		baseFileName = strings.TrimSuffix(b, filepath.Ext(b))
	} else {
		var t toolbox.Tools
		baseFileName = t.RandomString(10)
	}

	err = v.Encoder.Engine.EncodeToHLS(v, baseFileName)
	if err != nil {
		return "", err
	}
	return baseFileName, nil
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
func (v *Video) encodeToMP4() (string, error) {
	// Make sure output directory exists.
	err := v.createDirIfNotExists()
	if err != nil {
		return "", err
	}

	baseFileName := ""

	if !v.Options.RenameOutput {
		// Get base filename.
		b := path.Base(v.InputFile)
		baseFileName = strings.TrimSuffix(b, filepath.Ext(b))
	} else {
		var t toolbox.Tools
		baseFileName = t.RandomString(10)
	}

	err = v.Encoder.Engine.EncodeToMP4(v, baseFileName)
	if err != nil {
		return "", err
	}

	return baseFileName, nil
}
