package cleanid3

import (
	"fmt"
	"io"
	"math"
	"os"
	"strings"

	"github.com/bogem/id3v2"
	"github.com/golang/glog"
	"github.com/pkg/errors"
)

// Frame interface unifying the basic functions we need accross frame
// types.
type Frame interface {
	// Log tag name and all important details for this frame.
	Log(string)
	// Update the tag with new frame information.
	UpdateTag(*id3v2.Tag, string, string)
	// Get displayed text we may need to clean.
	FrameText() string
	// Remove the frame from that tag.
	Delete(*id3v2.Tag, string)
}

//
// TextFrame.
//

// TextFrame pulls in the id3v2-package type to make it extendable.
type TextFrame id3v2.TextFrame

// FF FE 44 00 49 00 53 00 43 00 49 00 44 00 00 00

// Log for TextFrame.
func (tf TextFrame) Log(k string) {
	glog.Infof("%s: %s (%s)", k, tf.Text, tf.Encoding.Name)
}

// UpdateTag for TextFrame.
func (tf TextFrame) UpdateTag(tag *id3v2.Tag, k string, value string) {
	tag.AddTextFrame(k, tf.Encoding, value)
}

// FrameText for TextFrame.
func (tf TextFrame) FrameText() string {
	return tf.Text
}

func (tf TextFrame) Delete(tag *id3v2.Tag, k string) {
	tag.DeleteFrames(k)
}

//
// UserDefinedTextFrame.
//

// UserDefinedTextFrame pulls in the id3v2-package type to make it extendable.
type UserDefinedTextFrame id3v2.UserDefinedTextFrame

// Log for UserDefinedTextFrame.
func (tf UserDefinedTextFrame) Log(k string) {
	glog.Infof("%s: %s: %s (%s)", k, tf.Description, tf.Value, tf.Encoding.Name)
}

// UpdateTag for UserDefinedTextFrame.
func (tf UserDefinedTextFrame) UpdateTag(tag *id3v2.Tag, k string, value string) {
	// These frames are not unique by key but additionally "subkeyed"
	// by their description. To update such frame, we need to remove all
	// of them and add them again as updated.
	frames := tag.GetFrames(k)
	tag.DeleteFrames(k)
	for _, f := range frames {
		udtf, _ := f.(id3v2.UserDefinedTextFrame)
		if tf.Description != udtf.Description {
			tag.AddUserDefinedTextFrame(udtf)
		} else {
			udtf = id3v2.UserDefinedTextFrame{
				Encoding:    tf.Encoding,
				Description: tf.Description,
				Value:       value,
			}
			tag.AddUserDefinedTextFrame(udtf)
		}
	}
}

// FrameText for UserDefinedTextFrame.
func (tf UserDefinedTextFrame) FrameText() string {
	return tf.Value
}

// Delete for UserDefinedTextFrame.
func (tf UserDefinedTextFrame) Delete(tag *id3v2.Tag, k string) {
	// These frames are not unique by key but additionally "subkeyed"
	// by their description. To remove such frame, we need to remove all
	// of them and add all but the one to delete again.
	frames := tag.GetFrames(k)
	tag.DeleteFrames(k)
	for _, f := range frames {
		udtf, _ := f.(id3v2.UserDefinedTextFrame)
		if tf.Description != udtf.Description {
			tag.AddUserDefinedTextFrame(udtf)
		}
	}
}

//
// CommentFrame.
//

// CommentFrame pulls in the id3v2-package type to make it extendable.
type CommentFrame id3v2.CommentFrame

// Log for CommentFrame.
func (cf CommentFrame) Log(k string) {
	message := fmt.Sprintf("%s: ", k)
	if len(cf.Description) > 0 {
		message += fmt.Sprintf("%s: ", cf.Description)
	}
	if len(cf.Language) > 0 {
		message += fmt.Sprintf("%s: ", cf.Language)
	}
	message += cf.Text
	glog.Info(message)
}

// UpdateTag for CommentFrame.
func (cf CommentFrame) UpdateTag(tag *id3v2.Tag, k string, value string) {
	newcf := id3v2.CommentFrame{
		Encoding:    cf.Encoding,
		Language:    cf.Language,
		Description: cf.Description,
		Text:        value,
	}
	tag.AddCommentFrame(newcf)
}

// FrameText for CommentFrame.
func (cf CommentFrame) FrameText() string {
	return cf.Text
}

func (cf CommentFrame) Delete(tag *id3v2.Tag, k string) {
	tag.DeleteFrames(k)
}

//
// UnsynchronisedLyricsFrame.
//

// UnsynchronisedLyricsFrame pulls in the id3v2-package type to make it extendable.
type UnsynchronisedLyricsFrame id3v2.UnsynchronisedLyricsFrame

// Log for UnsynchronisedLyricsFrame.
func (uslf UnsynchronisedLyricsFrame) Log(k string) {
	message := fmt.Sprintf("%s: ", k)
	if len(uslf.ContentDescriptor) > 0 {
		message += fmt.Sprintf("%s: ", uslf.ContentDescriptor)
	}
	if len(uslf.Language) > 0 {
		message += fmt.Sprintf("%s: ", uslf.Language)
	}
	message += uslf.Lyrics
	glog.Info(message)
}

// UpdateTag for UnsynchronisedLyricsFrame.
func (uslf UnsynchronisedLyricsFrame) UpdateTag(tag *id3v2.Tag, k string, value string) {
	newuslf := id3v2.UnsynchronisedLyricsFrame{
		Encoding:          uslf.Encoding,
		Language:          uslf.Language,
		ContentDescriptor: uslf.ContentDescriptor,
		Lyrics:            value,
	}
	tag.AddUnsynchronisedLyricsFrame(newuslf)
}

// FrameText for UnsynchronisedLyricsFrame.
func (uslf UnsynchronisedLyricsFrame) FrameText() string {
	return uslf.Lyrics
}

func (uslf UnsynchronisedLyricsFrame) Delete(tag *id3v2.Tag, k string) {
	tag.DeleteFrames(k)
}

//
//
//

// Clean input string.
//
// Removes additions if tainted by useless information.
func cleanedString(forbiddenWords []string, value string) (bool, string) {
	// Left most dirty word location.
	smallestIndex := math.MaxInt32

	for _, dirt := range forbiddenWords {
		index := strings.Index(value, dirt)

		if index != -1 {
			if index < smallestIndex {
				smallestIndex = index
			}
		}
	}

	if smallestIndex == math.MaxInt32 {
		return false, value
	}

	// Those dirty tag additions commonly are trailing
	// useful data, assume we only need to remove the
	// dirt from the tail.
	runes := []rune(value)
	return true, strings.TrimSpace(string(runes[:smallestIndex]))
}

func checkForID3V1(file string) (int, error) {
	glog.Info("Checking for ID3V1 tag")

	f, err := os.Open(file)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to open file '%s'", file)
	}

	stat, err := f.Stat()
	if err != nil {
		return 0, errors.Wrap(err, "failed to get file info")
	}

	// Exit early when the file is too small for a complete ID3V1.
	if stat.Size() < 128 {
		return 0, nil
	}

	buf := make([]byte, 3)

	n, err := f.Read(buf)
	if err != nil || n < len(buf) {
		return 0, errors.Wrap(err, "failed to read possible TAG at head")
	}

	if buf[0] == 'T' && buf[1] == 'A' && buf[2] == 'G' {
		glog.Info("ID3V1 tag found at head")
		return 1, nil
	}

	_, err = f.Seek(-128, 2)
	if err != nil {
		return 0, errors.Wrap(err, "failed to seek to possible TAG at tail")
	}

	n, err = f.Read(buf)
	if err != nil || n < len(buf) {
		return 0, errors.Wrap(err, "failed to read possible TAG at tail")
	}

	if buf[0] == 'T' && buf[1] == 'A' && buf[2] == 'G' {
		glog.Info("ID3V1 tag found at tail")
		return 2, nil
	}

	return 0, nil
}

func removeID3V1(file string, whence int) error {
	glog.Infof("Removing ID3V1 from %s\n", file)

	originalFile, err := os.Open(file)
	if err != nil {
		return errors.Wrapf(err, "failed to open file '%s'", file)
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

	if whence == 1 {
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

		if readBytes > 0 && whence == 2 {
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

// Clean parses the ID3V2 tags from the given file, removes occurrences of
// forbidden words in any text frame, update the file if needed. Additionally
// we check for existing ID3V1 tags and simply remove them altogether - it is
// 2021.
func Clean(words []string, file string, dryRun bool) error {
	glog.Infof("Processing %s\n", file)

	// Open file and parse tag in it.
	tag, err := id3v2.Open(file, id3v2.Options{Parse: true})
	if err != nil {
		return errors.Wrapf(err, "ID3V2 parsing failed for '%s'", file)
	}
	defer tag.Close()

	isFileDirty := false

	for k, s := range tag.AllFrames() {
		// Any cleanable frame.
		if k[0] == 'T' || k == "COMM" || k == "USLT" {
			for _, f := range s {
				var frame Frame

				if k == "COMM" {
					// "COMM".
					cf, _ := f.(id3v2.CommentFrame)
					frame = CommentFrame(cf)
				} else if k == "USLT" {
					// "USLT".
					uslf, _ := f.(id3v2.UnsynchronisedLyricsFrame)
					frame = UnsynchronisedLyricsFrame(uslf)
				} else if k == "TXXX" {
					// "TXXX".
					tf, _ := f.(id3v2.UserDefinedTextFrame)
					frame = UserDefinedTextFrame(tf)
				} else {
					// Any text frame that is not "TXXX".
					tf, _ := f.(id3v2.TextFrame)
					frame = TextFrame(tf)
				}

				frame.Log(k)

				cleaned, value := cleanedString(words, frame.FrameText())

				if cleaned {
					if len(value) > 0 {
						fmt.Printf("Updating %s: %s\n", k, value)
						frame.UpdateTag(tag, k, value)
					} else {
						fmt.Printf("Removing frame %s\n", k)
						frame.Delete(tag, k)
					}
					isFileDirty = true
				}
			}
		} else if k[0] == 'W' {
			// Any URL frame "W***".
			glog.Infof("%s: ????", k)
			fmt.Printf("Removing frame %s\n", k)
			tag.DeleteFrames(k)
			isFileDirty = true
		} else {
			// Any other frame id.
			glog.Infof("%s: ????", k)
		}
	}

	if isFileDirty {
		if !dryRun {
			glog.Info("Saving cleaned file")
			if err = tag.Save(); err != nil {
				return errors.Wrap(err, "failed to save ID3V2 tags")
			}
		} else {
			glog.Info("Skipping save for dry run")
		}
	} else {
		glog.Info("File was clean already")
	}

	foundID3V1At, err := checkForID3V1(file)
	if err != nil {
		return errors.Wrap(err, "failed to check for ID3V1")
	}

	if foundID3V1At != 0 {
		if !dryRun {
			fmt.Println("Removing ID3V1")
			err = removeID3V1(file, foundID3V1At)
			if err != nil {
				return errors.Wrap(err, "failed to remove ID3V1")
			}
		} else {
			glog.Info("Skipping ID3V1 removal for dry run")
		}
	}

	return nil
}
