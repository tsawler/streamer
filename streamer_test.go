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
		specifyEncoder bool
	}{
		{name: "test_new", args: args{jobQueue: make(chan VideoProcessingJob, 10)}, specifyEncoder: false},
		{name: "test_new_with_encoder", args: args{jobQueue: make(chan VideoProcessingJob, 10)}, specifyEncoder: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got *VideoDispatcher
			if tt.specifyEncoder {
				var engine testEncoder
				e := Processor{
					Engine: &engine,
				}
				got = New(tt.args.jobQueue, tt.args.maxWorkers, e)
			} else {
				got = New(tt.args.jobQueue, tt.args.maxWorkers)
			}

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
			v := wp.NewVideo(tt.args.id, "./a/b.mp4", "./output", tt.args.enc, testNotifyChan, tt.args.ops)
			if v.Options.RenameOutput != tt.args.ops.RenameOutput {
				t.Errorf("wrong value for rename; got %t expected %t", v.Options.RenameOutput, tt.args.ops.RenameOutput)
			}
		})
	}
}

func Test_encodeToMP4(t *testing.T) {
	type args struct {
		id  int
		enc string
		ops *VideoOptions
	}
	tests := []struct {
		name          string
		processor     Processor
		expectSuccess bool
		args          args
	}{
		{name: "mp4", processor: testProcessor, expectSuccess: true, args: args{1, "mp4", &VideoOptions{RenameOutput: false}}},
		{name: "mp4 rename", processor: testProcessor, expectSuccess: true, args: args{2, "mp4", &VideoOptions{RenameOutput: true}}},
		{name: "mp4 no ops", processor: testProcessor, expectSuccess: true, args: args{3, "mp4", nil}},
		{name: "mp4 failing", processor: testProcessorFailing, expectSuccess: false, args: args{4, "mp4", &VideoOptions{RenameOutput: false}}},
		{name: "mp4 rename failing", processor: testProcessorFailing, expectSuccess: false, args: args{5, "mp4", &VideoOptions{RenameOutput: true}}},
		{name: "mp4 no ops failing", processor: testProcessorFailing, expectSuccess: false, args: args{6, "mp4", nil}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wp := New(make(chan VideoProcessingJob), 1)
			wp.Processor = tt.processor
			v := wp.NewVideo(tt.args.id, "./testdata/i.mp4", "./testdata/output", "mp4", testNotifyChan, tt.args.ops)

			err := v.encodeToMP4()
			if err != nil && tt.expectSuccess {
				t.Errorf("%s: encode to mp4 failed: %s", tt.name, err.Error())
				return
			}

			if err == nil && !tt.expectSuccess {
				t.Errorf("%s: encode to mp4 did not fail, and it should", tt.name)
				return
			}

			// We only wait for a channel for successful encodes, since we are testing
			// the encoder, and not the encode() function.
			if tt.expectSuccess {
				result := <-testNotifyChan
				if !result.Successful && tt.expectSuccess {
					t.Errorf("%s: encoding failed", tt.name)
				}
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
		name          string
		args          args
		expectSuccess bool
	}{
		{name: "mp4", args: args{1, "mp4", &VideoOptions{RenameOutput: false}}, expectSuccess: true},
		{name: "hls rename", args: args{2, "hls", &VideoOptions{RenameOutput: true}}, expectSuccess: true},
		{name: "hls encrypted", args: args{3, "hls-encrypted", &VideoOptions{RenameOutput: true}}, expectSuccess: true},
		{name: "expect error", args: args{4, "fish", &VideoOptions{RenameOutput: true}}, expectSuccess: false},
	}
	for _, tt := range tests {
		wp := New(make(chan VideoProcessingJob), 1)
		v := wp.NewVideo(tt.args.id, "./testdata/i.mp4", "./testdata/output", tt.args.enc, testNotifyChan, tt.args.ops)
		v.Encoder = testProcessor

		v.encode()

		result := <-testNotifyChan
		if result.Successful != tt.expectSuccess {
			t.Errorf("%s: encoding failed: %s", tt.name, result.Message)
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

			v := wp.NewVideo(tt.args.id, "./a/b.mp4", "./output", tt.args.enc, testNotifyChan, tt.args.ops)

			videoQueue <- VideoProcessingJob{Video: v}

			result := <-testNotifyChan
			if !result.Successful {
				t.Errorf("%s: encoding failed", tt.name)
			}
		})
	}
}
