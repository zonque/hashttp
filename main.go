package main

import (
  "golang.org/x/sys/unix"
  "crypto/sha512"
  "encoding/binary"
  "encoding/hex"
  "flag"
  "fmt"
  "io"
  "log"
  "net/http"
  "os"
  "path"
  "strconv"
  "syscall"
  "unsafe"
)

type ImageReader struct {
  fileName string
  fileType string
  hashSum string
  totalSize int64
}

func alignTo(value int64, alignment int64) int64 {
  return (value + alignment - 1) & ^(alignment - 1)
}

func (reader *ImageReader) DetermineType(file *os.File) string {
  // SquashFS

  type SquashFsHeader struct {
    Smagic int32
    Inodes int32
    MkfsZime int32
    BlockSize int32
    Fragments int32
    Compression int16
    BlockLog int16
    Flags int16
    NoIds int16
    Smajor int16
    Sminor int16
    RootInode int64
    BytesUsed int64
    /* ignore the rest */
  }

  header := SquashFsHeader{}
  file.Seek(0, io.SeekStart)
  binary.Read(file, binary.LittleEndian, &header)

  if header.Smagic == 0x73717369 {
    reader.totalSize = alignTo(header.BytesUsed, 4096)
    return "squashfs"
  }

  stat, err := file.Stat()
  if err == nil {
    switch stat.Mode() & os.ModeType {
    case 0:
      // regular file
      reader.totalSize = stat.Size()
      return "plain"

    case os.ModeDevice:
      // block device - try to determine total size
      _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), unix.BLKGETSIZE64, uintptr(unsafe.Pointer(&reader.totalSize)))
      if errno == 0 {
        return "block-device"
      }
    }
  }

  return "unknown"
}

func (reader *ImageReader) Open(name string) error {
  file, err := os.OpenFile(name, os.O_RDONLY, 0)

  if err != nil {
    return err
  }

  defer file.Close()
  reader.fileName = name
  reader.fileType = reader.DetermineType(file)

  hash := sha512.New()
  file.Seek(0, io.SeekStart)
  io.CopyN(hash, file, reader.totalSize)
  reader.hashSum = hex.EncodeToString(hash.Sum(nil))

  return nil
}

func (reader *ImageReader) httpHandler(writer http.ResponseWriter, request *http.Request) {
  file, err := os.OpenFile(reader.fileName, os.O_RDONLY, 0)

  if err != nil {
    http.Error(writer, err.Error(), 500)
    log.Print(err)
    return
  }

  defer file.Close()
  log.Print("Handling request from " + request.RemoteAddr + " for " + reader.fileName + " - " + request.URL.Path[1:])
  io.CopyN(writer, file, reader.totalSize)
  writer.Header().Set("Content-Length", strconv.FormatInt(reader.totalSize, 10))
}

type sourcesFlags []string

func (i *sourcesFlags) String() string {
  return ""
}

func (i *sourcesFlags) Set(value string) error {
  *i = append(*i, value)
  return nil
}

var sources sourcesFlags

func contains(s []string, e string) bool {
  for _, a := range s {
    if a == e {
      return true
    }
  }

  return false
}

func main() {
  flag.Var(&sources, "source", "squashfs source to serve")
  port := flag.Int("port", 0, "port to listen to")
  url_prefix := flag.String("url-prefix", "", "Prefix for HTTP URLs")
  flag.Parse()

  if len(sources) == 0 || *port == 0 {
    log.Fatal("At least one source and a port are required.")
  }

  var matches []string

  for _, source := range sources {
    reader := ImageReader{}

    log.Print("Processing source " + source + " ...")

    err := reader.Open(source)
    if err != nil {
      log.Fatal(err)
    }

    match := path.Clean(path.Join("/", *url_prefix, reader.hashSum))

    if contains(matches, match) {
      log.Print("Ignoring " + source + " which has a hash that was already seen")
    } else {
      log.Print("Serving " + source + " (type " + reader.fileType + ") on " + match)
      http.HandleFunc(match, reader.httpHandler)
      matches = append(matches, match)
    }
  }

  log.Printf("Listening on port %d ...", *port)
  err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
  if err != nil {
    log.Fatal(err)
  }
}
