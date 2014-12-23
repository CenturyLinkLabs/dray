package main

import (
	"github.com/CenturyLinkLabs/stevedore/api"
)

func main() {
	api.ListenAndServe()
}

// For streaming responses
//type flushWriter struct {
//flusher http.Flusher
//writer  io.Writer
//}

//func newFlushWriter(writer io.Writer) flushWriter {
//fw := flushWriter{writer: writer}
//if flusher, ok := writer.(http.Flusher); ok {
//fw.flusher = flusher
//}

//return fw
//}

//func (fw *flushWriter) Write(p []byte) (n int, err error) {
//n, err = fw.writer.Write(p)
//if fw.flusher != nil {
//fw.flusher.Flush()
//}
//return
//}
