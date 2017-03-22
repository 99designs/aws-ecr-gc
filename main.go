package main

import (
	"fmt"
	"strings"

	"github.com/99designs/aws-ecr-gc/gc"
	"github.com/99designs/aws-ecr-gc/model"
	"github.com/99designs/aws-ecr-gc/registry"
)

// TODO: CLI flags
var region = "us-east-1"
var repo = "workbench-ci"
var keepCounts map[string]uint = map[string]uint{
	"release-production": 26,
	"build":              26,
}

func main() {
	ecr := registry.NewSession(region)
	images, err := ecr.Images(repo)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Total images in %s (%s): %d\n", repo, region, len(images))
	gcParams := gc.Params{KeepCounts: keepCounts, DeleteUntagged: true}
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
