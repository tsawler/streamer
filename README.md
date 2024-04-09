<a href="https://golang.org"><img src="https://img.shields.io/badge/powered_by-Go-3362c2.svg?style=flat-square" alt="Built with GoLang"></a>
[![License](http://img.shields.io/badge/license-mit-blue.svg?style=flat-square)](https://raw.githubusercontent.com/tsawler/streamer/master/LICENSE)

# Streamer

Streamer is a simple package which creates a worker pool to encode videos to web-ready format. 
Currently, streamer encodes to MP4, HLS, and HLS encrypted formats.

## Requirements

Streamer requires [ffmpeg](https://ffmpeg.org/) to be in your path.

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

	// Get a new streamer (worker pool) with three workers. We will push things to 
	// encode to the videoQueue channel, and get our results back on the notifyChan.
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

## Encoding to HLS encrypted

First, generate a key and vector (requires OpenSSL):

```
> openssl rand 16 > enc.key  # Saves an encryption to encrypt the video to a file named enc.key.
> openssl rand -hex 16       # Generates a vector (prints to terminal)
> de0efc88a53c730aa764648e545e3874
```

Create a file named enc.keyinfo:
```
https://whatever.com/enc.key
enc.key
de0efc88a53c730aa764648e545e3874
```

*Note*: enc.key **must** be at the root level of the project that uses this
package. The location of enc.keyinfo can be specified when creating
a variable of `streamer.Video` type, in the field `streamer.VideoOptions`, e.g.

~~~go
// Create the options for this encode.
ops := &streamer.VideoOptions{
    RenameOutput:    false,
    Secret:          "enc.key",
    KeyInfo:         "./keys/enc.keyinfo", // Specify path to enc.keyinfo
    SegmentDuration: 10,
}

// Get the video by calling NewVideo on the worker pool object.
myVideo := wp.NewVideo(4, "./upload/myvid.mp4", "./output", "hls-encrypted", notifyChan, ops)
~~~