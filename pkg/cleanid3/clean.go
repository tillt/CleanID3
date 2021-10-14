package cleanid3

import (
	"fmt"
	"math"
	"strings"

	"github.com/bogem/id3v2/v2"
	"github.com/golang/glog"
	"github.com/pkg/errors"
)

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

	frames := tag.AllFrames()
	for k, s := range frames {
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
			for _ = range s {
				glog.Infof("%s: ????", k)
			}
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

	return nil
}
