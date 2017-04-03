// package model provides entities like Image which are decoupled from
// third-party libraries like aws-sdk-go.
package model

import (
	"sort"
	"time"
)

// type Image provides the attributes of an ECR image required to decide
// upon and execute deletion.
type Image struct {
	Digest   string
	PushedAt time.Time
	Tags     []string
}

// type Images is a slice of Image capable of providing a sorted copy.
type Images []Image

// NewestFirst returns a copy of images sorted most recently pushed first.
func (images Images) CopyNewestFirst() Images {
	result := make(Images, len(images))
	copy(result, images)
	sort.Slice(result, func(i, j int) bool {
		a := result[i].PushedAt
		b := result[j].PushedAt
		return a.After(b)
	})
	return result
}

// type ImageID represents the identity of an image, either by tag or digest.
type ImageID struct {
	Digest string
	Tag    string
}

// type ImageFailure represents a failure to perform an operation on an image.
type ImageFailure struct {
	ID     ImageID
	Code   string
	Reason string
}

// type DeleteImagesResult holds the success and failure details of a
// BatchDeleteImage operation.
type DeleteImagesResult struct {
	Deletions []ImageID
	Failures  []ImageFailure
}
