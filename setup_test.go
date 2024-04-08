package streamer

import (
	"os"
	"testing"
)

var testProcessor Processor
var testNotifyChan chan ProcessingMessage

func TestMain(m *testing.M) {
	var te testEncoder
	testProcessor = Processor{
		Engine: &te,
	}
	testNotifyChan = make(chan ProcessingMessage, 10)
	os.Exit(m.Run())
}

type testEncoder struct{}

// EncodeToMP4 takes a Video object and a base file name, and encodes to MP4 format.
func (ve *testEncoder) EncodeToMP4(v *Video, baseFileName string) error {
	return nil
}

// EncodeToHLS takes a Video object and a base file name, and encodes to HLS format.
func (ve *testEncoder) EncodeToHLS(v *Video, baseFileName string) error {
	return nil
}

// EncodeToHLSEncrypted takes a Video object and a base file name, and encodes to encrypted HLS format.
func (ve *testEncoder) EncodeToHLSEncrypted(v *Video, baseFileName string) error {
	return nil
}
