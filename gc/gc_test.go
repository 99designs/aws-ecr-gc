package gc_test

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/99designs/aws-ecr-gc/gc"
	"github.com/99designs/aws-ecr-gc/model"
)

func TestNoDeletions(t *testing.T) {
	epoch := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	images := model.Images{
		img(epoch, 0, "a", "foo-1"),
		img(epoch, 1, "b", "foo-2"),
		img(epoch, 2, "c", "foo-3"),
		img(epoch, 3, "d", "foo-4"),
		img(epoch, 4, "e", "bar"),
		img(epoch, 5, "f", "bar-backup"),
		img(epoch, 6, "g", "baz"),
		img(epoch, 7, "g", "baz-other"),
		img(epoch, 8, "h"),
	}
	result := gc.ImagesToDelete(images, gc.Params{
		KeepCounts: map[string]uint{
			"foo": 4,
			"bar": 2,
		},
		DeleteUntagged: false,
	})
	if len(result) != 0 {
		t.Error("Expected zero deletions")
	}
}

func TestWithDeletions(t *testing.T) {
	epoch := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	images := model.Images{
		img(epoch, 0, "a", "foo-1"),      // GC: old foos
		img(epoch, 1, "b", "foo-2"),      // GC: old foos
		img(epoch, 2, "c", "foo-3"),      // keep: recent foo
		img(epoch, 3, "d", "foo-4"),      // keep: recent foo
		img(epoch, 4, "e", "bar"),        // GC: old bar
		img(epoch, 5, "f"),               // GC: untagged
		img(epoch, 6, "g", "bar"),        // GC: old bar
		img(epoch, 7, "h", "bar-a"),      // GC: old bar
		img(epoch, 8, "i", "bar-b"),      // keep: recent bar
		img(epoch, 8, "j", "bar-c"),      // keep: recent bar
		img(epoch, 9, "k", "baz"),        // keep: unmanaged tag
		img(epoch, 10, "l", "baz-other"), // keep: unmanaged tag
		img(epoch, 11, "m"),              // GC: untagged
	}
	result := gc.ImagesToDelete(images, gc.Params{
		KeepCounts: map[string]uint{
			"foo": 2,
			"bar": 2,
		},
		DeleteUntagged: true,
	})
	var digests []string
	for _, img := range result {
		digests = append(digests, img.Digest)
	}
	expected := []string{"m", "h", "g", "f", "e", "b", "a"}
	if !reflect.DeepEqual(digests, expected) {
		t.Errorf(
			"expected deletion of %s; got %s",
			strings.Join(expected, ","),
			strings.Join(digests, ","),
		)
	}
}

func img(epoch time.Time, offset time.Duration, digest string, tags ...string) model.Image {
	return model.Image{
		Digest:   digest,
		Tags:     tags,
		PushedAt: epoch.Add(offset * time.Hour),
	}
}
