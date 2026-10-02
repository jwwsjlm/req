package compress

import (
	"io"

	"github.com/molecule-man/go-brrr"
)

// BrotliReader 延迟解压 br 响应，底层响应体的所有权随 Reader 一起关闭。
type BrotliReader struct {
	Body io.ReadCloser // underlying Response.Body
	br   *brrr.Reader  // lazily-initialized brotli reader
}

func NewBrotliReader(body io.ReadCloser) *BrotliReader {
	return &BrotliReader{Body: body}
}

// Read 在首次调用时建立解码器，让 transport 有机会先包装底层 Body。
func (br *BrotliReader) Read(p []byte) (n int, err error) {
	if br.br == nil {
		br.br = brrr.NewReader(br.Body)
	}
	return br.br.Read(p)
}

// Close 释放解码缓冲并关闭底层 Body，保留底层关闭错误。
func (br *BrotliReader) Close() error {
	if br.br != nil {
		br.br.Close()
	}
	return br.Body.Close()
}

func (br *BrotliReader) GetUnderlyingBody() io.ReadCloser {
	return br.Body
}

func (br *BrotliReader) SetUnderlyingBody(body io.ReadCloser) {
	br.Body = body
}
