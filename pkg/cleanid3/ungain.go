package cleanid3

import (
	"fmt"

	"github.com/bogem/id3v2/v2"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"golang.org/x/exp/slices"
)

func Ungain(file string, dryRun bool) error {
	glog.Infof("Processing %s\n", file)

	tag, err := id3v2.Open(file, id3v2.Options{Parse: true})
	if err != nil {
		return errors.Wrapf(err, "ID3V2 parsing failed for '%s'", file)
	}
	defer tag.Close()

	isFileDirty := false

	frames := tag.AllFrames()
	for k, s := range frames {
		if k == "TXXX" {
			for _, f := range s {
				var frame Frame
				tf, _ := f.(id3v2.UserDefinedTextFrame)
				frame = UserDefinedTextFrame(tf)

				frame.Log(k)

				gainDescriptions := []string{
					"replaygain_album_gain",
					"replaygain_album_peak",
					"replaygain_reference_loudness",
					"replaygain_track_gain",
					"replaygain_track_peak",
					"rgain:track",
					"rgain:album",
					"MP3GAIN_ALBUM_MINMAX",
					"MP3GAIN_MINMAX",
					"MP3GAIN_UNDO",
				}

				gainer := slices.Contains(gainDescriptions, tf.Description)
				if gainer {
					fmt.Printf("Removing frame %s:%s\n", k, tf.Description)
					frame.Delete(tag, k)
					isFileDirty = true
				}
			}
		}
	}

	if isFileDirty {
		if !dryRun {
			glog.Info("Saving ungained file")
			if err = tag.Save(); err != nil {
				return errors.Wrap(err, "failed to save ID3V2 tags")
			}
		} else {
			glog.Info("Skipping save for dry run")
		}
	} else {
		glog.Info("File was without gain already")
	}

	return nil
}
