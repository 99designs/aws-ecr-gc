package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/99designs/aws-ecr-gc/gc"
	"github.com/99designs/aws-ecr-gc/model"
	"github.com/99designs/aws-ecr-gc/registry"
)

type keepCountMap map[string]uint

func (k keepCountMap) String() string {
	return fmt.Sprintf("%#v", k)
}

func (k keepCountMap) Set(value string) error {
	parts := strings.SplitN(value, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("expected prefix=COUNT e.g. release=4")
	}
	prefix := parts[0]
	count, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return fmt.Errorf("expected N in %s=N to be non-negative integer", prefix)
	}
	k[prefix] = uint(count)
	return nil
}

func main() {
	var region string
	var repo string
	var deleteUntagged bool
	keepCounts := keepCountMap{}
	flag.StringVar(&region, "region", os.Getenv("AWS_DEFAULT_REGION"), "AWS region")
	flag.StringVar(&repo, "repo", "", "AWS ECR repository name")
	flag.BoolVar(&deleteUntagged, "delete-untagged", deleteUntagged, "whether to delete untagged images")
	flag.Var(&keepCounts, "keep", "map of image tag prefixes to how many to keep, e.g. --keep release=4 --keep build=8")
	flag.Parse()
	if region == "" || repo == "" {
		flag.Usage()
		os.Exit(2)
	}

	ecr := registry.NewSession(region)
	images, err := ecr.Images(repo)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Total images in %s (%s): %d\n", repo, region, len(images))

	gcParams := gc.Params{KeepCounts: keepCounts, DeleteUntagged: deleteUntagged}
	deletionList := gc.ImagesToDelete(images, gcParams)
	printImages("Images to delete", deletionList)
	result, err := ecr.DeleteImages(repo, deletionList)
	if err != nil {
		panic(err)
	}
	printResult(result)
}

func printImages(heading string, images model.Images) {
	fmt.Printf("%s (%d)\n", heading, len(images))
	for _, img := range images {
		fmt.Printf(
			"  %s: %s... [%s]\n",
			img.PushedAt.Format("2006-01-02 15:04:05"),
			img.Digest[0:16],
			strings.Join(img.Tags, ", "),
		)
	}
}

func printResult(result *model.DeleteImagesResult) {
	fmt.Printf("Deleted (%d)\n", len(result.Deletions))
	for _, id := range result.Deletions {
		fmt.Printf("  %s... (%s)\n", id.Digest[0:16], id.Tag)
	}
	fmt.Printf("Failures (%d)\n", len(result.Failures))
	for _, f := range result.Failures {
		fmt.Printf("  %s... %s: %s\n", f.Id.Digest[0:16], f.Code, f.Reason)
	}
}
