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
	notifyChan := make(chan streamer.ProcessingMessage, 10)
	defer close(notifyChan)

	// Create a channel to send work to.
	videoQueue := make(chan streamer.VideoProcessingJob, 10)
	defer close(videoQueue)

	// Get a new streamer (worker pool).
	wp := streamer.New(videoQueue, 3)

	// Start the worker pool.
	wp.Run()

	// Create a video that converts mp4 to web ready mp4.
	video := wp.NewVideo(1, "./upload/puppy1.mp4", "./output", "mp4", notifyChan, nil)

	// Create a second video object that should fail, since input is not a valid video file.
	video2 := wp.NewVideo(2, "./upload/i.srt", "./output", "mp4", notifyChan, nil)

	// Create a third video object, encoding to HLS.
	video3 := wp.NewVideo(3, "./upload/puppy2.mp4", "./output", "hls", notifyChan, nil)

	// Create a fourth video object, encoding to HLS encrypted.
	ops := &streamer.VideoOptions{
		RenameOutput:    true,
		Secret:          "enc.key",
		KeyInfo:         "./keys/enc.keyinfo",
		SegmentDuration: 10,
	}
	video4 := wp.NewVideo(4, "./upload/puppy2.mp4", "./output", "hls-encrypted", notifyChan, ops)

	log.Println("Starting encode.")

	// Send videos to worker pool via channel.
	videoQueue <- streamer.VideoProcessingJob{Video: video}
	videoQueue <- streamer.VideoProcessingJob{Video: video2}
	videoQueue <- streamer.VideoProcessingJob{Video: video3}
	videoQueue <- streamer.VideoProcessingJob{Video: video4}

	log.Println("Waiting for results...")

	for i := 0; i < 4; i++ {
		msg := <-notifyChan
		log.Println("i:", i, "msg:", msg)
	}

	fmt.Println("Done!")
}
~~~