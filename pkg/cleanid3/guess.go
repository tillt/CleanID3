package cleanid3

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/bogem/id3v2/v2"
	"github.com/golang/glog"
	"github.com/pkg/errors"
)

// Meta describes the most rudimentary information we care for in the process of enhancing ID3 tags.
type Meta struct {
	album  string
	artist string
	title  string
	genre  string
	year   int
	track  int
	tracks int
	disc   int
	discs  int
}

func guessEnum(e string) (int, int, error) {
	v1 := 0
	v2 := 0

	r, err := regexp.Compile("([0-9])*(//?([0-9])*)?")
	if err != nil {
		return v1, v2, errors.Wrap(err, "failed to compile regexp")
	}

	matches := r.FindAllStringSubmatch(e, -1)

	if len(matches) == 0 {
		return v1, v2, errors.Errorf("index '%s' with unexpected format", e)
	}

	if len(matches[0]) > 1 {
		v1, err = strconv.Atoi(matches[0][1])
		if err != nil {
			return v1, v2, errors.Wrap(err, "failed to convert index string into number")
		}
	}

	// Try to get an optional item count.
	if len(matches[0]) > 3 {
		v2, err = strconv.Atoi(matches[0][3])
		if err != nil {
			v2 = 0
		}
	}

	return v1, v2, nil
}

func read(file string) (*Meta, error) {
	// Open file and parse tag in it.
	tag, err := id3v2.Open(file, id3v2.Options{Parse: true})
	if err != nil {
		return nil, errors.Wrapf(err, "ID3V2 parsing failed for '%s'", file)
	}
	defer tag.Close()

	meta := new(Meta)

	frames := tag.AllFrames()
	for k, s := range frames {
		// Any cleanable frame.
		if k[0] != 'T' {
			continue
		}
		for _, f := range s {
			var frame Frame

			tf, _ := f.(id3v2.TextFrame)
			frame = TextFrame(tf)

			if k == "TALB" {
				meta.album = frame.FrameText()
			} else if k == "TIT2" {
				meta.title = frame.FrameText()
			} else if k == "TPE1" {
				meta.artist = frame.FrameText()
			} else if k == "TCON" {
				meta.genre = frame.FrameText()
			} else if k == "TPOS" {
				meta.disc, meta.discs, err = guessEnum(frame.FrameText())
				if err != nil {
					glog.Errorf("failed to guess disc index and count: %w", err)
					meta.disc = 0
					meta.discs = 0
				}
			} else if k == "TRCK" {
				meta.track, meta.tracks, err = guessEnum(frame.FrameText())
				if err != nil {
					glog.Errorf("failed to guess track index and count: %w", err)
					meta.track = 0
					meta.tracks = 0
				}
			} else if k == "TYER" {
				meta.year, err = strconv.Atoi(frame.FrameText())
				if err != nil {
					meta.year = 0
				}
			}

		}
	}

	glog.Infof("read ID3 from '%s' and found %+v'", file, meta)

	return meta, nil
}

func guess(path string) (*Meta, error) {
	glog.Infof("Guessing %s\n", path)

	parts := strings.Split(path, "/")

	file := parts[len(parts)-1]

	dots := strings.Split(file, ".")
	if len(dots) > 0 {
		file = strings.Join(dots[:len(dots)-1], ".")
	}

	// Get a complete title candidate.
	title := strings.TrimSpace(file)

	meta := new(Meta)

	// Try to extract "artist - title".
	f := strings.Split(title, "-")
	if len(f) >= 2 {
		meta.artist = strings.TrimSpace(f[0])
		meta.title = strings.TrimSpace(strings.Join(f[1:], "-"))
	} else {
		meta.title = strings.TrimSpace(f[0])
	}

	// Try to extract an index and maybe a count.
	r, err := regexp.Compile("([0-9])*:?([0-9]*)?[-,.: ]?(.*)")
	if err != nil {
		return nil, err
	}

	matches := r.FindAllStringSubmatch(meta.title, -1)
	if len(matches) > 0 {
		if len(matches[0]) > 2 {
			if len(matches[0][1]) > 0 {
				meta.track, err = strconv.Atoi(matches[0][1])
				if err != nil {
					meta.track = 0
				}
			}
			if len(matches[0][2]) > 0 {
				meta.tracks, err = strconv.Atoi(matches[0][2])
				if err != nil {
					meta.tracks = 0
				}
			}
			// Use what was not track index to continue parsing.
			meta.title = strings.TrimSpace(matches[0][3])
		}
	}

	meta.title = strings.Trim(meta.title, ":;,.- ")
	meta.artist = strings.Trim(meta.artist, ":;,.- ")

	// Try to extract album from parent folder name.
	if len(parts) > 1 {
		tempAlbum := parts[len(parts)-2]
		if len(tempAlbum) > 0 {
			// Lame way of excluding anything most certainly non album name.
			if !strings.Contains(tempAlbum, "MP3ADD") &&
				!strings.Contains(tempAlbum, "Downloads") &&
				!strings.Contains(tempAlbum, "tmp.") {
				meta.album = tempAlbum
			}
		}
	}

	matches = r.FindAllStringSubmatch(meta.album, -1)
	if len(matches) > 0 {
		if len(matches[0]) > 2 {
			if len(matches[0][1]) > 0 {
				meta.disc, err = strconv.Atoi(matches[0][1])
				if err != nil {
					meta.disc = 0
				}
			}
			if len(matches[0][2]) > 0 {
				meta.discs, err = strconv.Atoi(matches[0][2])
				if err != nil {
					meta.discs = 0
				}
			}
			// Use what was not track index to continue parsing.
			meta.album = strings.TrimSpace(matches[0][3])
		}
	}

	meta.album = strings.Trim(meta.album, ":;,.- ")

	glog.Infof("Guessed from path '%s' and found %+v'\n", path, meta)

	return meta, nil
}

// Enhance parses the ID3 metadata from a file and if important data
// like "Artist, Title or Album" are missing, tries to enhance those
// from information extracted from the path.
func Enhance(file string, dry bool) error {
	glog.Infof("Enhancing %s\n", file)

	// Read what the file gets us in important metadata from the path.
	meta, err := guess(file)
	if err != nil {
		glog.Error(err)
	}

	// Read the metadata from ID3.
	id3, err := read(file)
	if err != nil {
		return errors.Wrapf(err, "read failed for '%s'", file)
	}

	tag, err := id3v2.Open(file, id3v2.Options{Parse: true})
	if err != nil {
		return errors.Wrapf(err, "ID3V2 parsing failed for '%s'", file)
	}
	defer tag.Close()

	isFileDirty := false

	if len(id3.title) == 0 && len(meta.title) > 0 {
		isFileDirty = true
		fmt.Printf("Adding title %s\n", meta.title)
		tag.AddTextFrame("TIT2", id3v2.EncodingUTF16, meta.title)
	}

	if len(id3.artist) == 0 && len(meta.artist) > 0 {
		isFileDirty = true
		fmt.Printf("Adding artist %s\n", meta.artist)
		tag.AddTextFrame("TPE1", id3v2.EncodingUTF16, meta.artist)
	}

	if id3.track == 0 && meta.track > 0 {
		isFileDirty = true
		tracks := id3.tracks
		if tracks == 0 && meta.tracks > 0 {
			tracks = meta.tracks
		}
		fmt.Printf("Adding track %d/%d\n", meta.track, tracks)
		tag.AddTextFrame("TRCK", id3v2.EncodingISO, fmt.Sprintf("%d/%d", meta.track, tracks))
	}

	if id3.disc == 0 && meta.disc > 0 {
		isFileDirty = true
		discs := id3.discs
		if discs == 0 && meta.discs > 0 {
			discs = meta.discs
		}
		fmt.Printf("Adding disc %d/%d\n", meta.disc, discs)
		tag.AddTextFrame("TPOS", id3v2.EncodingISO, fmt.Sprintf("%d/%d", meta.disc, discs))
	}

	if len(id3.album) == 0 && len(meta.album) > 0 {
		isFileDirty = true
		fmt.Printf("Adding album %s\n", meta.album)
		tag.AddTextFrame("TALB", id3v2.EncodingUTF16, meta.album)
	}

	if isFileDirty {
		if !dry {
			glog.Info("Saving enhanced metadata")
			if err = tag.Save(); err != nil {
				return errors.Wrap(err, "failed to save ID3V2 tags")
			}
		} else {
			glog.Info("Skipping enhanced save for dry run")
		}
	} else {
		glog.Info("File did not need additional ID3 tagging")
	}

	return nil
}
