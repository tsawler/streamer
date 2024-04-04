# Streamer

## key info
Generate a key and vector:

```
openssl rand 16 > enc.key  # Key to encrypt the video
openssl rand -hex 16       # IV
# de0efc88a53c730aa764648e545e3874
```

Create enc.keyinfo:
```
https://whatever.com/enc.key
enc.key
de0efc88a53c730aa764648e545e3874
```

*Note*: enc.key **must** be at the root level of the project that uses this
package. The location of enc.keyinfo can be specified when creating
a `streamer.VideoProcessor` object.

## Sample usage

~~~go
package main

import (
	"fmt"
	"github.com/tsawler/streamer"
	"log"
)

func main() {
	// Create a channel to receive notifications.
	notifyChan := make(chan streamer.ProcessingMessage)
	defer close(notifyChan)

	// Create a channel to send work to.
	videoQueue := make(chan streamer.VideoProcessingJob, 10)
	defer close(videoQueue)

	// Get a new streamer (worker pool).
	wp := streamer.New(videoQueue, 2)

	// Start the worker pool.
	wp.Run()

	// Create a Video object for processing.
	video := streamer.Video{
		ID:              1,                // Arbitrary id of video.
		InputFile:       "./upload/j.mp4", // Where is the file to encode?
		OutputDir:       "./output",       // Where to create output file(s).
		SegmentDuration: 10,               // Duration of segments, in seconds (hls & hls-encrypted only).
		NotifyChan:      notifyChan,       // The channel to send notifications to.
		EncodingType:    "hls",            // Can be hls, mp4, or hls-encrypted.
		MaxRate1080p:    "2400k",
		MaxRate720p:     "1200k",
		MaxRate480p:     "800k",
	}
	//
	// Create a second Video object for processing.
	video2 := streamer.Video{
		ID:           2,
		InputFile:    "./upload/k.mp4",
		OutputDir:    "./output",
		NotifyChan:   notifyChan,
		EncodingType: "mp4",
		RenameOutput: true,
	}

	// Create a third video object that should fail, since
	// input is not a valid video file.
	video3 := streamer.Video{
		ID:           3,
		InputFile:    "./upload/i.srt",
		OutputDir:    "./output",
		NotifyChan:   notifyChan,
		EncodingType: "mp4",
	}

	// Create a fourth video object that's encrypted.
	video4 := streamer.Video{
		ID:           3,
		InputFile:    "./upload/j.mp4",
		OutputDir:    "./output",
		NotifyChan:   notifyChan,
		EncodingType: "hls-encrypted",
		KeyInfo:      "./keys/enc.keyinfo",
		Secret:       "enc.key",
		MaxRate1080p: "2400k",
		MaxRate720p:  "1200k",
		MaxRate480p:  "800k",
	}

	log.Println("Starting encode.")

	// Send videos to worker pool via channel.
	videoQueue <- streamer.VideoProcessingJob{Video: video}
	videoQueue <- streamer.VideoProcessingJob{Video: video2}
	videoQueue <- streamer.VideoProcessingJob{Video: video3}
	videoQueue <- streamer.VideoProcessingJob{Video: video4}

	log.Println("Waiting for results...")

	for i := 0; i < 4; i++ {
		msg := <-notifyChan
		fmt.Printf("Video ID #%d finished: %s.\n", msg.ID, msg.Message)
		fmt.Printf("Output file for video ID #%d: %s.\n", msg.ID, msg.OutputFile)
	}

	fmt.Println("Done!")
}
~~~