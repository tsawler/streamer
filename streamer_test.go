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
		name           string
		args           args
		useFailEncoder bool
	}{
		{name: "test_new", args: args{jobQueue: make(chan VideoProcessingJob, 10), maxWorkers: 1}, useFailEncoder: false},
		{name: "test_new_with_encoder", args: args{jobQueue: make(chan VideoProcessingJob, 10), maxWorkers: 3}, useFailEncoder: true},
		{name: "test_new_with_encoder", args: args{jobQueue: make(chan VideoProcessingJob, 10), maxWorkers: 0}, useFailEncoder: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got *VideoDispatcher
			if tt.useFailEncoder {
				var engine testEncoder
				e := Processor{
					Engine: &engine,
				}
				got = New(tt.args.jobQueue, tt.args.maxWorkers, e)
			} else {
				got = New(tt.args.jobQueue, tt.args.maxWorkers)
			}

			if got.maxWorkers != tt.args.maxWorkers && tt.args.maxWorkers > 0 {
				t.Errorf("New() = %d, want %d", got.maxWorkers, tt.args.maxWorkers)
			}

			if got.maxWorkers != 1 && tt.args.maxWorkers == 0 {
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
		id  int
		enc string
		ops *VideoOptions
	}
	tests := []struct {
		name string
		args args
	}{
		{name: "mp4", args: args{1, "mp4", &VideoOptions{RenameOutput: false}}},
		{name: "mp4 rename", args: args{1, "mp4", &VideoOptions{RenameOutput: true}}},
		{name: "hls", args: args{1, "hls", &VideoOptions{RenameOutput: false}}},
		{name: "hls rename", args: args{1, "hls", &VideoOptions{RenameOutput: true}}},
		{name: "mp4 empty", args: args{1, "", &VideoOptions{RenameOutput: false}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wp := New(make(chan VideoProcessingJob), 1)
			v := wp.NewVideo(tt.args.id, "./a/b.mp4", "./testdata/output", tt.args.enc, testNotifyChan, tt.args.ops)
			if v.Options.RenameOutput != tt.args.ops.RenameOutput {
				t.Errorf("wrong value for rename; got %t expected %t", v.Options.RenameOutput, tt.args.ops.RenameOutput)
			}
		})
	}
}

func Test_encode(t *testing.T) {
	type args struct {
		id  int
		enc string
		ops *VideoOptions
	}
	tests := []struct {
		name           string
		args           args
		useFailEncoder bool
		expectSuccess  bool
	}{
		{name: "mp4", args: args{1, "mp4", &VideoOptions{RenameOutput: false}}, expectSuccess: true, useFailEncoder: false},
		{name: "mp4_no_options", args: args{2, "mp4", nil}, expectSuccess: true, useFailEncoder: false},
		{name: "mp4_fail", args: args{3, "mp4", &VideoOptions{RenameOutput: false}}, expectSuccess: false, useFailEncoder: true},
		{name: "hls rename", args: args{4, "hls", &VideoOptions{RenameOutput: true}}, expectSuccess: true, useFailEncoder: false},
		{name: "hls_fail", args: args{5, "hls", &VideoOptions{RenameOutput: true}}, expectSuccess: false, useFailEncoder: true},
		{name: "hls encrypted", args: args{6, "hls-encrypted", &VideoOptions{RenameOutput: false}}, expectSuccess: true, useFailEncoder: false},
		{name: "hls encrypted rename", args: args{7, "hls-encrypted", &VideoOptions{RenameOutput: true}}, expectSuccess: true, useFailEncoder: false},
		{name: "hls encrypted_fail", args: args{8, "hls-encrypted", &VideoOptions{RenameOutput: true}}, expectSuccess: false, useFailEncoder: true},
		{name: "invalid encoding type", args: args{9, "fish", &VideoOptions{RenameOutput: true}}, expectSuccess: false},
	}

	for _, tt := range tests {
		wp := New(make(chan VideoProcessingJob), 1)
		v := wp.NewVideo(tt.args.id, "./testdata/i.mp4", "./testdata/output", tt.args.enc, testNotifyChan, tt.args.ops)
		if tt.useFailEncoder {
			v.Encoder = testProcessorFailing
		} else {
			v.Encoder = testProcessor
		}

		v.encode()

		result := <-testNotifyChan
		if result.Successful != tt.expectSuccess {
			t.Errorf("%s: expected result.Successful of %t but got %t", tt.name, tt.expectSuccess, result.Successful)
		}
	}
}

func Test_pool(t *testing.T) {
	type args struct {
		id  int
		enc string
		ops *VideoOptions
	}
	tests := []struct {
		name string
		args args
	}{
		{name: "mp4", args: args{1, "mp4", &VideoOptions{RenameOutput: false}}},
		{name: "mp4 rename", args: args{1, "mp4", &VideoOptions{RenameOutput: true}}},
		{name: "hls", args: args{1, "hls", &VideoOptions{RenameOutput: false}}},
		{name: "hls rename", args: args{1, "hls", &VideoOptions{RenameOutput: true}}},
		{name: "mp4 empty", args: args{1, "", &VideoOptions{RenameOutput: false}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			videoQueue := make(chan VideoProcessingJob, 10)
			wp := New(videoQueue, 3)
			wp.Processor = testProcessor
			wp.Run()

			v := wp.NewVideo(tt.args.id, "./a/b.mp4", "./testdata/output", tt.args.enc, testNotifyChan, tt.args.ops)

			videoQueue <- VideoProcessingJob{Video: v}

			result := <-testNotifyChan
			if !result.Successful {
				t.Errorf("%s: encoding failed", tt.name)
			}
		})
	}
}
