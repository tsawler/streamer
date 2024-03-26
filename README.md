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
	notifyChan := make(chan streamer.ProcessingMessage)
	videoQueue := make(chan streamer.VideoProcessingJob, 10)
	defer close(videoQueue)

	wp := streamer.New(videoQueue, 2)
	wp.Run()

	video := streamer.Video{
		ID:              1,
		InputFile:       "./upload/k.mp4",
		OutputDir:       "./output",
		SegmentDuration: 10,
		NotifyChan:      notifyChan,
		EncodingType:    "hls",
	}

	video2 := streamer.Video{
		ID:              2,
		InputFile:       "./upload/j.mp4",
		OutputDir:       "./output",
		SegmentDuration: 10,
		NotifyChan:      notifyChan,
		EncodingType:    "mp4",
	}

	log.Println("Starting encode")
	videoQueue <- streamer.VideoProcessingJob{
		Video: video,
	}

	videoQueue <- streamer.VideoProcessingJob{
		Video: video2,
	}

	log.Println("Waiting for result...")

	for i := 0; i < 2; i++ {
		msg := <-notifyChan
		fmt.Println("Done", msg.ID, msg.Message)
	}

	fmt.Println("Done!")
}
~~~