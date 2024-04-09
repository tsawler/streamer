package streamer

import "testing"

func Test_All_Encoders(t *testing.T) {
	type args struct {
		id  int
		enc string
	}
	tests := []struct {
		name string
		args args
	}{
		{name: "mp4", args: args{id: 1, enc: "mp4"}},
		{name: "mp4", args: args{id: 2, enc: "hls"}},
		{name: "hls-encrypted", args: args{id: 3, enc: "hls-encrypted"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wp := New(make(chan VideoProcessingJob), 1)
			v := wp.NewVideo(tt.args.id, "./a/b.mp4", "./output", tt.args.enc, testNotifyChan, nil)

			v.encode()

			result := <-testNotifyChan
			if result.Successful {
				t.Errorf("%s: expected result.Successful of false but got %t", tt.name, result.Successful)
			}
		})
	}

}
