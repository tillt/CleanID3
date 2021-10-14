package cleanid3

import (
	"io"
	"os"

	"github.com/golang/glog"
	"github.com/pkg/errors"
)

const (
	// ID3V1TagAtUnknown signals that the tag location is not known yet or not found.
	ID3V1TagAtUnknown int = 0
	// ID3V1TagAtFront signals that the tag is located at front of file.
	ID3V1TagAtFront int = 1
	// ID3V1TagAtTail signals that the tag is located at tail of file.
	ID3V1TagAtTail int = 2
)

// CheckForID3V1 tries to locate an ID3V1 tag within the given file.
func CheckForID3V1(file string) (int, error) {
	glog.Infof("Checking %s for ID3V1 tag\n", file)

	f, err := os.Open(file)
	if err != nil {
		return ID3V1TagAtUnknown, errors.Wrapf(err, "failed to open file '%s'", file)
	}

	stat, err := f.Stat()
	if err != nil {
		return ID3V1TagAtUnknown, errors.Wrap(err, "failed to get file info")
	}

	// Exit early when the file is too small for a complete ID3V1.
	if stat.Size() < 128 {
		return ID3V1TagAtUnknown, nil
	}

	buf := make([]byte, 3)

	n, err := f.Read(buf)
	if err != nil || n < len(buf) {
		return ID3V1TagAtUnknown, errors.Wrap(err, "failed to read possible TAG at head")
	}

	if buf[0] == 'T' && buf[1] == 'A' && buf[2] == 'G' {
		glog.Info("ID3V1 tag found at head")
		return ID3V1TagAtFront, nil
	}

	_, err = f.Seek(-128, 2)
	if err != nil {
		return ID3V1TagAtUnknown, errors.Wrap(err, "failed to seek to possible TAG at tail")
	}

	n, err = f.Read(buf)
	if err != nil || n < len(buf) {
		return ID3V1TagAtUnknown, errors.Wrap(err, "failed to read possible TAG at tail")
	}

	if buf[0] == 'T' && buf[1] == 'A' && buf[2] == 'G' {
		glog.Info("ID3V1 tag found at tail")
		return ID3V1TagAtTail, nil
	}

	return ID3V1TagAtUnknown, nil
}

// RemoveID3V1 removes the ID3V1 tag from the given file.
func RemoveID3V1(file string, whence int) error {
	glog.Infof("Removing ID3V1 from %s\n", file)

	var err error

	if whence == ID3V1TagAtUnknown {
		whence, err = CheckForID3V1(file)
		if err != nil {
			return errors.Wrapf(err, "failed to check '%s' for ID3V1", file)
		}
	}
	if whence == ID3V1TagAtUnknown {
		return nil
	}

	originalFile, err := os.Open(file)
	if err != nil {
		return errors.Wrapf(err, "failed to open '%s'", file)
	}

	originalStat, err := originalFile.Stat()
	if err != nil {
		return errors.Wrap(err, "failed to get file info")
	}

	name := file + "-id3v1"
	newFile, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE, originalStat.Mode())
	if err != nil {
		return errors.Wrapf(err, "failed to create destination file '%s'", name)
	}
	defer func() {
		os.Remove(newFile.Name())
	}()

	buf := make([]byte, 128*1024)

	if whence == ID3V1TagAtFront {
		_, err = originalFile.Seek(128, 0)
		if err != nil {
			return errors.Wrap(err, "failed to skip ID3V1 header at head")
		}
	}

	for {
		readBytes, err := originalFile.Read(buf)
		if err != nil && err != io.EOF {
			return errors.Wrap(err, "failed to read source data")
		}

		if readBytes > 0 && whence == ID3V1TagAtTail {
			offset, err := originalFile.Seek(0, 1)
			if err != nil {
				return errors.Wrap(err, "failed to get source file position")
			}
			if offset > originalStat.Size()-128 {
				if readBytes < 128 {
					break
				}
				readBytes -= 128
			}
		}

		if readBytes == 0 {
			break
		}

		_, err = newFile.Write(buf[:readBytes])
		if err != nil {
			return errors.Wrap(err, "failed to write data to destination")
		}
	}

	os.Remove(originalFile.Name())

	err = os.Rename(newFile.Name(), originalFile.Name())
	if err != nil {
		return errors.Wrap(err, "failed to rename temporary file")
	}

	return nil
}
