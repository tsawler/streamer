package streamer

import (
	"errors"
	"os"
	"testing"
)

// testProcessor is a variable which satisfies the Encoder interface, and always returns no error.
var testProcessor Processor

// testProcessorFailing is a variable which satisfies the Encoder interface, and always returns an error.
var testProcessorFailing Processor

// testNotifyChan is the channel we use to get the results of an encode, for testing.
var testNotifyChan chan ProcessingMessage

func TestMain(m *testing.M) {
	var te testEncoder
	testProcessor = Processor{
		Engine: &te,
	}
	var tef testEncoderFailing
	testProcessorFailing = Processor{
		Engine: &tef,
	}
	testNotifyChan = make(chan ProcessingMessage, 10)
	os.Exit(m.Run())
}

// testEncoder is a type which satisfies the Encoder interface. We use it to
// test successful encoding, so all its methods return nil (no error).
type testEncoder struct{}

// EncodeToMP4 takes a Video object and a base file name, and simulates encoding to MP4 format successfully.
func (te *testEncoder) EncodeToMP4(v *Video, baseFileName string) error {
	return nil
}

// EncodeToHLS takes a Video object and a base file name, and simulates encoding to HLS format successfully.
func (te *testEncoder) EncodeToHLS(v *Video, baseFileName string) error {
	return nil
}

// EncodeToHLSEncrypted takes a Video object and a base file name, and simulates encoding to encrypted HLS format
// successfully.
func (te *testEncoder) EncodeToHLSEncrypted(v *Video, baseFileName string) error {
	return nil
}

// testEncoderFailing is a type which satisfies the Encoder interface. We use it to
// test for encodes which fail, so all its methods return an error.
type testEncoderFailing struct{}

// EncodeToMP4 takes a Video object and a base file name, and simulates encoding to MP4 format unsuccessfully.
func (tef *testEncoderFailing) EncodeToMP4(v *Video, baseFileName string) error {
	return errors.New("some error")
}

// EncodeToHLS takes a Video object and a base file name, and simulates encoding to HLS format unsuccessfully.
func (tef *testEncoderFailing) EncodeToHLS(v *Video, baseFileName string) error {
	return errors.New("some error")
}

// EncodeToHLSEncrypted takes a Video object and a base file name, and simulates encoding to encrypted HLS format
// unsuccessfully.
func (tef *testEncoderFailing) EncodeToHLSEncrypted(v *Video, baseFileName string) error {
	return errors.New("some error")
}
