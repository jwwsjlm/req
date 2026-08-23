# 上传与下载

## 文件路径、内存和 Reader 上传

```go
resp, err := client.R().
	SetFile("file", "./report.pdf").
	Post(uploadURL)
```

内存内容：

```go
resp, err := client.R().
	SetFileBytes("file", "note.txt", []byte("hello")).
	Post(uploadURL)
```

Reader：

```go
resp, err := client.R().
	SetFileReader("file", "note.txt", strings.NewReader("hello")).
	Post(uploadURL)
```

需要显式 part Content-Type 时使用 `SetMultipartField(paramName, filename, contentType, reader)`。

## 完全自定义 FileUpload

```go
upload := req.FileUpload{
	ParamName:   "file",
	FileName:    "report.bin",
	FileSize:    size,
	ContentType: "application/octet-stream",
	GetFileContent: func() (io.ReadCloser, error) {
		return os.Open(path)
	},
}

resp, err := client.R().SetFileUpload(upload).Post(uploadURL)
```

`GetFileContent` 应在每次调用时返回新的 reader，以支持发送和可能的重试。已知 `FileSize` 且 `ContentType` 可确定时，multipart 可以计算 `Content-Length`；无法确定长度时使用 chunked 传输。

## 表单字段和多个文件

```go
resp, err := client.R().
	SetFormData(map[string]string{"kind": "invoice"}).
	SetFiles(map[string]string{
		"front": "./front.png",
		"back":  "./back.png",
	}).
	Post(uploadURL)
```

## 上传进度

```go
resp, err := client.R().
	SetUploadCallback(func(info req.UploadInfo) {
		log.Printf("%s: %d/%d", info.FileName, info.UploadedSize, info.FileSize)
	}).
	SetFile("file", path).
	Post(uploadURL)
```

`SetUploadCallback` 使用默认最小间隔；`SetUploadCallbackWithInterval` 可调整。启用上传回调会强制 chunked encoding，因此目标服务必须接受 chunked。回调应轻量、不可长时间阻塞。

## 下载到文件或 Writer

```go
resp, err := client.R().
	SetOutputFile("./downloads/archive.zip").
	Get(downloadURL)
```

统一目录：

```go
client.SetOutputDirectory("./downloads")
resp, err := client.R().SetOutputFile("archive.zip").Get(downloadURL)
```

Writer：

```go
var output bytes.Buffer
resp, err := client.R().SetOutput(&output).Get(downloadURL)
```

下载进度使用 `SetDownloadCallback` / `SetDownloadCallbackWithInterval`。输出到文件/writer 时避免再把完整 body 作为普通响应结果使用。

## 手工流式读取

```go
resp, err := client.R().
	DisableAutoReadResponse().
	Get(downloadURL)
if err != nil {
	return err
}
defer resp.Body.Close()

_, err = io.Copy(destination, resp.Body)
```

手工读取必须关闭 body；读到 EOF 有利于连接复用。仍可通过 `SetMaxResponseSize` 对流式读取设置上限。

## 并行分片下载

```go
err := client.NewParallelDownload(downloadURL).
	SetOutputFile("archive.zip").
	SetConcurrency(8).
	SetSegmentSize(16 << 20).
	Do(ctx)
```

服务端需要支持 `Range`，且 HEAD 能提供可靠 `Content-Length`。并发数并非越高越好，应考虑服务端限流、磁盘吞吐和连接预算。

完整本地示例：[upload_download_test.go](examples/upload_download_test.go)。
