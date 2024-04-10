package streamer

import "testing"

func Test_All_Encoders(t *testing.T) {
	type args struct {
		id   int
		file string
		enc  string
	}
	tests := []struct {
		name          string
		expectSuccess bool
		args          args
	}{
		{name: "mp4 error", expectSuccess: false, args: args{id: 1, file: "a.mp4", enc: "mp4"}},
		{name: "hls error", expectSuccess: false, args: args{id: 2, file: "a.mp4", enc: "hls"}},
		{name: "hls-encrypted error", expectSuccess: false, args: args{id: 3, file: "a.mp4", enc: "hls-encrypted"}},
		{name: "mp4", expectSuccess: true, args: args{id: 4, file: "./testdata/dog.mp4", enc: "mp4"}},
		{name: "hls", expectSuccess: true, args: args{id: 5, file: "./testdata/dog.mp4", enc: "hls"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wp := New(make(chan VideoProcessingJob), 1)
			v := wp.NewVideo(tt.args.id, tt.args.file, "./testdata/output", tt.args.enc, testNotifyChan, nil)

			v.encode()

			result := <-testNotifyChan
			if result.Successful != tt.expectSuccess {
				t.Log(result.Message)
				t.Errorf("%s: expected result of %t but got %t", tt.name, tt.expectSuccess, result.Successful)
			}
		})
	}

}
