package main

import (
	"flag"
	"github.com/CenturyLinkLabs/stevedore/api"
)

func main() {
	port := flag.Int("p", 2000, "specify the port on which the server will run")
	flag.Parse()
	api.ListenAndServe(*port)
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
