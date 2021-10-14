# CleanID3

Parses ID3V2 tags from the supplied file, removes occurrences of forbidden words from text frames, removes URL frames, updates the file if needed.

## Background

Consider your MP3 collection was having a colorful past and you want to make sure things look more consistent. ID3 tags often get exploited by conversion tools and distribution platforms. Those vandals add useless traces of their own existence - URLs, product names, etc...

## Configuration

Just add those nasty traces you want removed to the list in `forbidden.txt`. The list comes pre-configured with some usual suspects.

### `forbidden.txt`

`forbidden.txt` contains a list of terms that are to be cleaned out.

For removing all URLs that from any ID3 tag, use something like this;

```
https://
http://
```

These defaults cause CleanID3 to modify every tag that contains a URL, removing it. CleanID3 will assume that anything following the forbidden word is also unwanted and cleans it. Text left of the forbidden word is left untouched.

For removing a bunch of known tags;
```
Tagged by:
This tag done with
converted by
```

## Build

```bash
go build
```

## Run

Running CleanID3 to process `file.mp3` while enabling verbose debug logging and using a forbidden words list from the current work directory `./forbidden.txt`;

```bash
./cleanid3 -verbose -forbidden=./forbidden.txt file.mp3
```

Running CleanID3 to process any file supplied via `stdin`;

```bash
./cleanid3
```

Verbose example output from a ID3-tagged file that contains a comment with a URL starting with "http://" which is part of the default `forbidden.txt`;

```
I0119 23:38:27.066641   59976 main.go:37] Processing test.mp3
[...]
I0119 23:38:27.074113   59976 main.go:102] COMM: http://www.example.net
Removing tag COMM
[...]
I0119 23:38:27.074133   59976 main.go:131] Saving cleaned file
```

The file gets updated with a removed ID3 comment.


## Application Example: iTunes Match Library Import

The following script makes use of cleanid3 for ensureing files added to the Music app are stripped of garbage tags. When files exceed the iTunes Match filesize limit of 200mb, we split the source into chunks of 60minutes and add those.

```bash
#!/bin/bash
set -e -x

cleanid3_bin="/Users/till/go/bin/cleanid3"
mp3splt_bin="/usr/local/bin/mp3splt"
max_filesize="200000000"
split_duration="60.0"

function add() {
  title=$(basename "$1")
  message=$($cleanid3_bin "$1" 2>&1)
  osascript<<EOSA_ADD
set foo to posix file "$1" as alias
display notification "$message" with title "Music Add" subtitle "♫ $title"
tell application "Music" to add foo
EOSA_ADD
}

for f in "$@"; do
  $(chmod 0644 "$f")
  size=$(stat -f%z "$f")
  if [ "$size" -gt "$max_filesize" ]; then
    workdir=$(mktemp -d)
    $mp3splt_bin -d "$workdir" -f -t "$split_duration" -a "$f"
    for ff in $workdir/*.mp3; do
     add "$ff"
    done
    rm -rf "$workdir"
  else
    add "$f"
  fi
done
```

For splitting MP3s, we use `mp3splt` - get it via homebrew;

```bash
brew install mp3splt
```
