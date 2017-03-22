package model

import (
	"sort"
	"time"
)

type Image struct {
	Digest   string
	PushedAt time.Time
	Tags     []string
}

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

type ImageId struct {
	Digest string
	Tag    string
}

type ImageFailure struct {
	Id     ImageId
	Code   string
	Reason string
}

type DeleteImagesResult struct {
	Deletions []ImageId
	Failures  []ImageFailure
}
