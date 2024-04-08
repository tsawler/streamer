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
		id      int
		enc     string
		max1080 string
		max740  string
		max480  string
		notify  chan ProcessingMessage
		rename  bool
	}
	tests := []struct {
		name string
		args args
	}{
		{name: "mp4", args: args{1, "mp4", "2400k", "1200k", "800k", make(chan ProcessingMessage), false}},
		{name: "hls", args: args{1, "hls", "", "", "", make(chan ProcessingMessage), true}},
		{name: "mp4 empty", args: args{1, "", "", "", "", make(chan ProcessingMessage), true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewVideo(tt.args.id, tt.args.enc, tt.args.max1080, tt.args.max740, tt.args.max480, tt.args.notify, tt.args.rename)
			if v.RenameOutput != tt.args.rename {
				t.Errorf("wrong value for rename; got %t expected %t", v.RenameOutput, tt.args.rename)
			}
		})
	}
}
