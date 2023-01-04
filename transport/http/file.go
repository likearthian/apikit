package http

import (
	"io"

	utype "github.com/likearthian/types"
)

type GetFileRequestDTO struct {
	Aud        string `query:"aud"`
	Descriptor string `query:"descriptor"`
}

type FileDescriptor struct {
	FileID string
	Expiry int64
}

type FilePayload struct {
	Content     []byte
	ContentType string
	FileName    string
}

func (fp *FilePayload) AddFile(name string, content []byte, contentType string) {
	fp.FileName = name
	fp.Content = content
	fp.ContentType = contentType
}

type FileStreamPayload struct {
	Reader      io.ReadCloser
	ContentType string
	FileName    string
}

func (fp *FileStreamPayload) AddFileStream(name string, reader io.ReadCloser, contentType string) {
	fp.FileName = name
	fp.Reader = reader
	fp.ContentType = contentType
}

type PostFileDTO struct {
	Content     []byte
	ContentType string
	FileName    string
	Path        string
	Aud         []string
}

type PreSubmitPQAReportDTO struct {
	Content        []byte
	ContentType    string
	FileName       string
	Sheet          string        `form:"sheet"`
	ProjectID      string        `form:"project"`
	TestType       string        `form:"tc_type"`
	UpdateExisting utype.Boolean `form:"update"`
}

func (s *PreSubmitPQAReportDTO) AddFile(name string, content []byte, contentType string) {
	s.FileName = name
	s.Content = content
	s.ContentType = contentType
}

type FileUploader[T any] interface {
	AddFile(name string, content []byte, contentType string)
	*T
}

type FileStreamUploader[T any] interface {
	AddFileStream(name string, reader io.ReadCloser, contentType string)
	*T
}

type SubmitReportRequestDTO struct {
	Ref string `query:"ref"`
}

type FileObjectDTO struct {
	ID                string        `json:"id"`
	ContentReadCloser io.ReadCloser `json:"-"`
	Filename          string        `json:"filename"`
	ContentType       string        `json:"content_type"`
	IsCompressed      bool          `json:"isCompressed"`
}

func (fob FileObjectDTO) IsOpen() bool {
	return fob.ContentReadCloser != nil
}

type FileResponse struct {
	Filename    string
	Content     io.ReadCloser
	ContentType string
}

type DownloadLinkDTO struct {
	Url string `json:"url"`
}
