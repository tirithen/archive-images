package main // import "github.com/tirithen/archive-images"
import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/mknote"
)

type MediaFile struct {
	Path      string
	CreatedAt time.Time
}

func (mediaFile *MediaFile) LoadMetaData() error {
	if mediaFile.HasMediaFileExtension() == false {
		return errors.New("Unable to verify that " + mediaFile.Path + " is a media file, it's either not supported or not a media file.")
	}

	file, err := os.Open(mediaFile.Path)
	if err != nil {
		return err
	}

	// First try EXIF
	exifData, err := exif.Decode(file)
	if err == nil {
		createdAt, err := exifData.DateTime()
		if err == nil {
			mediaFile.CreatedAt = createdAt
			return nil
		}
	}

	/* Skip for now as it does not provide creation date
	// Then try video/audio reader
	tagData, err := tag.ReadFrom(file)
	if err != nil {
		return errors.New("Unable to parse " + mediaFile.Path + ", this file is either not supported or not a media file.")
	}*/

	// Try to get creation date from filename
	datePattern := regexp.MustCompile(`(\d{4})-?(\d{2})-?(\d{2})[T\s_]*(\d{2}):?(\d{2}):?(\d{2}):?`)
	matches := datePattern.FindAllStringSubmatch(mediaFile.Path, -1)
	if len(matches) == 1 && len(matches[0]) == 7 {
		// TODO: fix timezone
		isoDate := matches[0][1] + "-" + matches[0][2] + "-" + matches[0][3] + "T" + matches[0][4] + ":" + matches[0][5] + ":" + matches[0][6]
		createdAt, err := time.Parse("2006-01-02T15:04:05", isoDate)

		if err == nil {
			mediaFile.CreatedAt = createdAt
			return nil
		}
	}

	return errors.New("Failed to load required meta data from " + mediaFile.Path)
}

func (mediaFile *MediaFile) Filename() string {
	return filepath.Base(mediaFile.Path)
}

func (mediaFile *MediaFile) CreatedDate() string {
	return mediaFile.CreatedAt.Format("2006-01-02")
}

func (mediaFile *MediaFile) HasMediaFileExtension() bool {
	// TODO: improve and extend this list (or find some other way)
	mediaFileExtensions := []string{".tiff", ".png", ".jpg", ".jpeg", ".mp4", ".avi", ".mov"}
	result := false

	for _, extension := range mediaFileExtensions {
		if result == false && strings.HasSuffix(mediaFile.Path, extension) == true {
			result = true
		}
	}

	return result
}

func getMediaFileListAtPath(directory string) (fileList []MediaFile, err error) {
	err = filepath.Walk(directory, func(path string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		mediaFile := MediaFile{Path: path}
		err = mediaFile.LoadMetaData()
		if err == nil {
			fileList = append(fileList, mediaFile)
		}

		return nil
	})

	return
}

func main() {
	exif.RegisterParsers(mknote.All...)

	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	fileList, err := getMediaFileListAtPath(pwd)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Found %d files to archive", len(fileList))

	for _, file := range fileList {
		dateDirectory := pwd + "/" + file.CreatedDate()
		if _, err := os.Stat(dateDirectory); os.IsNotExist(err) {
			os.MkdirAll(dateDirectory, os.ModePerm)
		}

		newPath := dateDirectory + "/" + file.Filename()
		if file.Path != newPath {
			os.Rename(file.Path, newPath)
		}
	}
}
