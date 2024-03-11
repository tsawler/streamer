package streamer

type Upload struct {
	FileToEncode string
}

type Processed struct {
	Path     string `json:"path"`
	FileName string `json:"file_name"`
}

type Encoder struct {
	PathToFFMpeg string
}

func New() *Encoder {
	return &Encoder{}
}
