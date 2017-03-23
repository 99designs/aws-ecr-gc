package gc

import (
	"strings"

	"github.com/99designs/aws-ecr-gc/model"
)

type Params struct {
	KeepCounts     map[string]uint
	DeleteUntagged bool
}

func ImagesToDelete(all model.Images, params Params) model.Images {
	var deletionList model.Images
	prefixes := knownPrefixes(params.KeepCounts)
	seenCounts := createSeenCounts(prefixes)
	for _, img := range all.CopyNewestFirst() {
		candidate := false
		recent := false
		untagged := false
		unmanaged := false
		if len(img.Tags) == 0 {
			untagged = true
		} else if hasUnknownTags(img.Tags, prefixes) {
			unmanaged = true
		}
		for _, t := range img.Tags {
			for p, keepCount := range params.KeepCounts {
				if strings.HasPrefix(t, p) {
					candidate = true
					if seenCounts[p] < keepCount {
						recent = true
					}
					seenCounts[p]++
				}
			}
		}
		if (untagged && params.DeleteUntagged) || (candidate && !recent && !unmanaged) {
			deletionList = append(deletionList, img)
		}
	}
	return deletionList
}

func createSeenCounts(prefixes []string) map[string]uint {
	seenCounts := make(map[string]uint)
	for _, p := range prefixes {
		seenCounts[p] = 0
	}
	return seenCounts
}

func knownPrefixes(keepCounts map[string]uint) []string {
	prefixes := make([]string, 0)
	for p, _ := range keepCounts {
		prefixes = append(prefixes, p)
	}
	return prefixes
}

func hasUnknownTags(tags []string, prefixes []string) bool {
	for _, t := range tags {
		match := false
		for _, p := range prefixes {
			if strings.HasPrefix(t, p) {
				match = true
				break
			}
		}
		if !match {
			return true
		}
	}
	return false
}
