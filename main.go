package main

import (
	"log"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
)

var repo = "workbench-ci"
var keepPrefixes []string = []string{"release-production", "build"}
var keepCount int = 16

type image struct {
	digest      string
	pushedAt    time.Time
	tags        []string
	gcCandidate bool
	gcKeep      bool
	gcUntagged  bool
}

func main() {
	log.Printf("Garbage collecting %s, keeping up to %d images each for tags prefixes:", repo, keepCount)
	for _, p := range keepPrefixes {
		log.Println("  ", p)
	}

	sess := session.Must(session.NewSession())
	client := ecr.New(sess, &aws.Config{Region: aws.String("us-east-1")})

	// all images, newest first
	images, err := describeImagesNewestFirst(client, repo)
	if err != nil {
		panic(err)
	}

	// mark candidates (has prefix) and keepers (first N per prefix)
	seen := make(map[string]int)
	for _, p := range keepPrefixes {
		seen[p] = 0
	}
	for i, img := range images {
		if len(img.tags) == 0 {
			images[i].gcUntagged = true
			images[i].gcCandidate = true
		} else {
			for _, t := range img.tags {
				for _, p := range keepPrefixes {
					if strings.HasPrefix(t, p) {
						images[i].gcCandidate = true
						if seen[p] < keepCount {
							images[i].gcKeep = true
						}
						seen[p]++
					}
				}
			}
		}
	}

	// count, prepare deletion list
	countCandidate := 0
	countIgnore := 0
	countKeep := 0
	countUntagged := 0
	deleteIds := make([]*ecr.ImageIdentifier, 0)
	for _, img := range images {
		if img.gcCandidate {
			countCandidate++
			if img.gcUntagged {
				countUntagged++
			}
			if img.gcKeep {
				countKeep++
			} else {
				d := img.digest // copy
				deleteIds = append(deleteIds, &ecr.ImageIdentifier{ImageDigest: &d})
			}
		} else {
			countIgnore++
		}
	}

	log.Printf(
		"Found %d images, %d untagged, %d ready for GC, %d retained (ignoring %d images with other tags)",
		len(images),
		countUntagged,
		countCandidate-countKeep,
		countKeep,
		countIgnore,
	)

	log.Printf("Deleting %d images", len(deleteIds))
	if len(deleteIds) > 0 {
		output, err := client.BatchDeleteImage(
			&ecr.BatchDeleteImageInput{
				ImageIds:       deleteIds,
				RepositoryName: &repo,
			},
		)
		if err != nil {
			panic(err)
		}
		if len(output.ImageIds) != 0 {
			log.Println("Deleted:")
			for _, deletedId := range output.ImageIds {
				log.Printf("  digest: %s, tag: %s", *deletedId.ImageDigest, *deletedId.ImageTag)
			}
		}
		if len(output.Failures) != 0 {
			log.Println("Failures:")
			for _, failure := range output.Failures {
				log.Println("  ", failure.String())
			}
		}
	}
}

func unpointerStrings(in []*string) []string {
	out := make([]string, 0)
	for _, s := range in {
		out = append(out, *s)
	}
	return out
}

func describeImagesNewestFirst(client *ecr.ECR, repo string) ([]image, error) {
	var err error
	images := make([]image, 0)
	listImagesInput := &ecr.ListImagesInput{
		RepositoryName: aws.String(repo),
	}
	listImagesPageNum := 0
	err = client.ListImagesPages(
		listImagesInput,
		func(page *ecr.ListImagesOutput, lastPage bool) bool {
			listImagesPageNum++
			log.Printf("ListImages page %d (%d images)\n", listImagesPageNum, len(page.ImageIds))
			describeImagesInput := &ecr.DescribeImagesInput{
				RepositoryName: aws.String(repo),
				ImageIds:       page.ImageIds,
			}
			describeImagesPageNum := 0
			err = client.DescribeImagesPages(
				describeImagesInput,
				func(page *ecr.DescribeImagesOutput, lastPage bool) bool {
					describeImagesPageNum++
					log.Printf("DescribeImages page %d (%d images)\n", describeImagesPageNum, len(page.ImageDetails))
					for _, img := range page.ImageDetails {
						images = append(images, image{
							digest:   *img.ImageDigest,
							pushedAt: *img.ImagePushedAt,
							tags:     unpointerStrings(img.ImageTags),
						})
					}
					return err == nil && describeImagesPageNum <= 100 // arbitrary terminator
				},
			)
			return err == nil && listImagesPageNum <= 100 // arbitrary terminator
		},
	)
	if err != nil {
		return nil, err
	}
	sort.Slice(images, func(i, j int) bool {
		a := images[i].pushedAt
		b := images[j].pushedAt
		return a.After(b)
	})
	return images, nil
}
