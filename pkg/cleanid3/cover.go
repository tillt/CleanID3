package cleanid3

import (
	"crypto/sha1"
	"encoding/hex"
	"io/ioutil"
	"path"
	"strings"

	"github.com/bogem/id3v2/v2"
	"github.com/golang/glog"
	"github.com/pkg/errors"
)

func CheckForCover(file string) (bool, error) {
	glog.Infof("Processing %s\n", file)

	hasCover := false

	// Open file and parse tag in it.
	tag, err := id3v2.Open(file, id3v2.Options{Parse: true})
	if err != nil {
		return hasCover, errors.Wrapf(err, "ID3V2 parsing failed for '%s'", file)
	}
	defer tag.Close()

	frames := tag.AllFrames()
	for k, s := range frames {
		// Any cleanable frame.
		if k == "APIC" {
			// APIC.
			for _, f := range s {
				pf, _ := f.(id3v2.PictureFrame)
				sha := sha1.Sum(pf.Picture)
				encoded := hex.EncodeToString(sha[:])
				hasCover = true

				glog.Infof("%s: type:%d mime:%s SHA:%s", k, pf.PictureType, pf.MimeType, encoded)
			}
		}
	}

	if hasCover {
		glog.Info("File has a cover")
	}

	return hasCover, nil
}

func AddCover(file, coverFile string) error {
	glog.Infof("Enhancing %s with cover\n", file)

	tag, err := id3v2.Open(file, id3v2.Options{Parse: true})
	if err != nil {
		return errors.Wrapf(err, "ID3V2 parsing failed for '%s'", file)
	}
	defer tag.Close()

	mimeType := "image/jpeg"

	coverFileExt := strings.ToLower(path.Ext(coverFile))

	if coverFileExt == ".png" {
		glog.Infof("Cover appears to be a PNG file\n")
		mimeType = "image/png"
	}

	contents, err := ioutil.ReadFile(coverFile)

	if err != nil {
		return errors.Wrapf(err, "Cover file read failed for '%s'", coverFile)
	}

	coverFrame := id3v2.PictureFrame{
		Encoding:    id3v2.EncodingUTF8,
		MimeType:    mimeType,
		PictureType: id3v2.PTFrontCover,
		Description: "Front cover",
		Picture:     contents,
	}

	tag.AddAttachedPicture(coverFrame)
	if err = tag.Save(); err != nil {
		return errors.Wrap(err, "failed to save ID3V2 tags with cover updates")
	}

	return nil
}
