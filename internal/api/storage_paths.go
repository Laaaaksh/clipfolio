package api

import "fmt"

// videoStorageDir is the object storage prefix holding everything for one
// video: its original upload, HLS renditions, master playlist, and
// thumbnail.
func videoStorageDir(videoID int64) string {
	return fmt.Sprintf("videos/%d", videoID)
}
