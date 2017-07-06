// package registry encapsulates access to AWS ECR, returning entities provided
// by package model.
package registry

import (
	"github.com/99designs/aws-ecr-gc/model"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
)

// type Session wraps a configured/authenticated ECR API session.
type Session struct {
	ecr *ecr.ECR
}

// func NewSession creates a Session for the given AWS region e.g. "us-east-1".
func NewSession(region string) *Session {
	sess := session.Must(session.NewSession())
	conf := aws.Config{Region: &region}
	return &Session{ecr: ecr.New(sess, &conf)}
}

// func Images returns a detailed list of all images in the specified
// repository.
func (s *Session) Images(repo string) (model.Images, error) {
	var handlerErr error
	var images model.Images
	var describeImagesPageNum, listImagesPageNum uint

	describeImagesPageHandler := func(page *ecr.DescribeImagesOutput, lastPage bool) bool {
		describeImagesPageNum++
		for _, img := range page.ImageDetails {
			images = append(images, imageFromAws(img))
		}
		return describeImagesPageNum <= 100 // arbitrary terminator
	}

	listImagesPageHandler := func(page *ecr.ListImagesOutput, lastPage bool) bool {
		listImagesPageNum++
		handlerErr = s.ecr.DescribeImagesPages(
			&ecr.DescribeImagesInput{RepositoryName: &repo, ImageIds: page.ImageIds},
			describeImagesPageHandler,
		)
		return handlerErr == nil && listImagesPageNum <= 100 // arbitrary terminator
	}

	err := s.ecr.ListImagesPages(
		&ecr.ListImagesInput{RepositoryName: &repo},
		listImagesPageHandler,
	)
	if err != nil {
		return nil, err
	}
	if handlerErr != nil {
		return nil, handlerErr
	}

	return images, nil
}

// func DeleteImages performs a BatchDeleteImage operation for the listed
// images in the specified repository.
func (s *Session) DeleteImages(repo string, images model.Images) (*model.DeleteImagesResult, error) {
	result := &model.DeleteImagesResult{}
	if len(images) == 0 {
		return result, nil
	}

	batchSize := 100
	var batches [][]*ecr.ImageIdentifier
	for start := 0; start < len(images); start += batchSize {
		var batch []*ecr.ImageIdentifier
		end := start + batchSize
		if end > len(images) {
			end = len(images)
		}

		for _, img := range images[start:end] {
			d := img.Digest
			batch = append(batch, &ecr.ImageIdentifier{ImageDigest: &d})
		}

		batches = append(batches, batch)
	}

	deleteBatch := func(ids []*ecr.ImageIdentifier) error {
		output, err := s.ecr.BatchDeleteImage(
			&ecr.BatchDeleteImageInput{
				ImageIds:       ids,
				RepositoryName: &repo,
			},
		)
		if err != nil {
			return err
		}
		for _, awsImgID := range output.ImageIds {
			imgID := model.ImageID{Digest: *awsImgID.ImageDigest, Tag: *awsImgID.ImageTag}
			result.Deletions = append(result.Deletions, imgID)
		}
		for _, awsFailure := range output.Failures {
			awsImgID := *awsFailure.ImageId
			imgID := model.ImageID{Digest: *awsImgID.ImageDigest, Tag: *awsImgID.ImageTag}
			failure := model.ImageFailure{ID: imgID, Code: *awsFailure.FailureCode, Reason: *awsFailure.FailureReason}
			result.Failures = append(result.Failures, failure)
		}
		return nil
	}

	for _, ids := range batches {
		err := deleteBatch(ids)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func imageFromAws(img *ecr.ImageDetail) model.Image {
	return model.Image{
		Digest:   *img.ImageDigest,
		PushedAt: *img.ImagePushedAt,
		Tags:     unpointerStrings(img.ImageTags),
	}
}

func unpointerStrings(in []*string) []string {
	out := make([]string, 0)
	for _, s := range in {
		out = append(out, *s)
	}
	return out
}
