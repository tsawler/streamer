package streamer

import (
	"errors"
	"os"
	"testing"
)

var testProcessor Processor
var testProcessorFailing Processor
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
	testNotifyChan = make(chan ProcessingMessage, 1)
	os.Exit(m.Run())
}

type testEncoder struct{}

// EncodeToMP4 takes a Video object and a base file name, and encodes to MP4 format.
func (te *testEncoder) EncodeToMP4(v *Video, baseFileName string) error {
	return nil
}

// EncodeToHLS takes a Video object and a base file name, and encodes to HLS format.
func (te *testEncoder) EncodeToHLS(v *Video, baseFileName string) error {
	return nil
}

// EncodeToHLSEncrypted takes a Video object and a base file name, and encodes to encrypted HLS format.
func (te *testEncoder) EncodeToHLSEncrypted(v *Video, baseFileName string) error {
	return nil
}

type testEncoderFailing struct{}

// EncodeToMP4 takes a Video object and a base file name, and encodes to MP4 format.
func (tef *testEncoderFailing) EncodeToMP4(v *Video, baseFileName string) error {
	return errors.New("some error")
}

// EncodeToHLS takes a Video object and a base file name, and encodes to HLS format.
func (tef *testEncoderFailing) EncodeToHLS(v *Video, baseFileName string) error {
	return errors.New("some error")
}

// EncodeToHLSEncrypted takes a Video object and a base file name, and encodes to encrypted HLS format.
func (tef *testEncoderFailing) EncodeToHLSEncrypted(v *Video, baseFileName string) error {
	return errors.New("some error")
}
