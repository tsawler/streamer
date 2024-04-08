package streamer

import (
	"reflect"
	"testing"
)

func TestNew(t *testing.T) {
	type args struct {
		jobQueue   chan VideoProcessingJob
		maxWorkers int
	}
	tests := []struct {
		name string
		args args
	}{
		{name: "test_new", args: args{jobQueue: make(chan VideoProcessingJob, 10)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New(tt.args.jobQueue, tt.args.maxWorkers)
			if got.maxWorkers != tt.args.maxWorkers {
				t.Errorf("New() = %d, want %d", got.maxWorkers, tt.args.maxWorkers)
			}
			var isChannel = reflect.ValueOf(got.jobQueue).Kind() == reflect.Chan
			if !isChannel {
				t.Error("jobQueue is not a channel")
			}
			channelType := reflect.ValueOf(got.jobQueue).Type().Elem()
			if channelType.Name() != "VideoProcessingJob" {
				t.Error("Incorrect channel type")
			}
		})
	}
}

func TestNewVideo(t *testing.T) {
	type args struct {
		id     int
		enc    string
		notify chan ProcessingMessage
		ops    *VideoOptions
	}
	tests := []struct {
		name string
		args args
	}{
		{name: "mp4", args: args{1, "mp4", make(chan ProcessingMessage), &VideoOptions{RenameOutput: false}}},
		{name: "hls", args: args{1, "hls", make(chan ProcessingMessage), &VideoOptions{RenameOutput: true}}},
		{name: "mp4 empty", args: args{1, "", make(chan ProcessingMessage), &VideoOptions{RenameOutput: false}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wp := New(make(chan VideoProcessingJob), 1)
			v := wp.NewVideo(tt.args.id, "./a/b.mp4", "./output", tt.args.enc, tt.args.notify, tt.args.ops)
			if v.Options.RenameOutput != tt.args.ops.RenameOutput {
				t.Errorf("wrong value for rename; got %t expected %t", v.Options.RenameOutput, tt.args.ops.RenameOutput)
			}
		})
	}
}

func Test_encodeToMP4(t *testing.T) {
	wp := New(make(chan VideoProcessingJob), 1)
	v := wp.NewVideo(1, "./testdata/i.mp4", "./testdata/output", "mp4", testNotifyChan, nil)
	v.Encoder = testProcessor

	err := v.encodeToMP4()
	if err != nil {
		t.Error("encode to mp4 failed", err)
		return
	}

	result := <-testNotifyChan
	if !result.Successful {
		t.Error("encoding failed")
	}
}
