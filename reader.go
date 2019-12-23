package main

import (
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

type imageReader struct {
	fileName  string
	fileType  string
	hashSum   string
	totalSize int64
}

func alignTo(value int64, alignment int64) int64 {
	return (value + alignment - 1) & ^(alignment - 1)
}

func (reader *imageReader) determineType(file *os.File) string {
	// SquashFS

	type SquashFsHeader struct {
		Smagic      int32
		Inodes      int32
		MkfsZime    int32
		BlockSize   int32
		Fragments   int32
		Compression int16
		BlockLog    int16
		Flags       int16
		NoIds       int16
		Smajor      int16
		Sminor      int16
		RootInode   int64
		BytesUsed   int64
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

func (reader *imageReader) open(name string) error {
	file, err := os.OpenFile(name, os.O_RDONLY, 0)

	if err != nil {
		return err
	}

	defer file.Close()
	reader.fileName = name
	reader.fileType = reader.determineType(file)

	hash := sha512.New()
	file.Seek(0, io.SeekStart)
	io.CopyN(hash, file, reader.totalSize)
	reader.hashSum = hex.EncodeToString(hash.Sum(nil))

	return nil
}

func (reader *imageReader) httpHandler(writer http.ResponseWriter, request *http.Request) {
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
