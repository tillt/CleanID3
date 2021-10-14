package cleanid3

import (
	"fmt"

	"github.com/bogem/id3v2/v2"
	"github.com/golang/glog"
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

// Delete for TextFrame.
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

		if id3v2.UserDefinedTextFrame(tf).UniqueIdentifier() != udtf.UniqueIdentifier() {
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
		if id3v2.UserDefinedTextFrame(tf).UniqueIdentifier() != udtf.UniqueIdentifier() {
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

// Delete for CommentFrame.
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

// Delete for UnsynchronisedLyricsFrame.
func (uslf UnsynchronisedLyricsFrame) Delete(tag *id3v2.Tag, k string) {
	tag.DeleteFrames(k)
}
