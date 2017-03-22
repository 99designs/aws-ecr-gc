package registry

import (
	"github.com/99designs/aws-ecr-gc/model"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
)

type Session struct {
	ecr *ecr.ECR
}

func NewSession(region string) *Session {
	sess := session.Must(session.NewSession())
	conf := aws.Config{Region: aws.String(region)}
	return &Session{ecr: ecr.New(sess, &conf)}
}

func (s *Session) Images(repo string) (model.Images, error) {
	var err error
	var images model.Images
	var describeImagesPageNum, listImagesPageNum uint

	describeImagesPageHandler := func(page *ecr.DescribeImagesOutput, lastPage bool) bool {
		describeImagesPageNum++
		for _, img := range page.ImageDetails {
			images = append(images, imageFromAws(img))
		}
		return err == nil && describeImagesPageNum <= 100 // arbitrary terminator
	}

	listImagesPageHandler := func(page *ecr.ListImagesOutput, lastPage bool) bool {
		listImagesPageNum++
		err = s.ecr.DescribeImagesPages(
			&ecr.DescribeImagesInput{RepositoryName: aws.String(repo), ImageIds: page.ImageIds},
			describeImagesPageHandler,
		)
		return err == nil && listImagesPageNum <= 100 // arbitrary terminator
	}

	err = s.ecr.ListImagesPages(
		&ecr.ListImagesInput{RepositoryName: aws.String(repo)},
		listImagesPageHandler,
	)
	if err != nil {
		return nil, err
	}

	return images, nil
}

func (s *Session) DeleteImages(repo string, images model.Images) (*model.DeleteImagesResult, error) {
	result := &model.DeleteImagesResult{}
	if len(images) == 0 {
		return result, nil
	}
	var ids []*ecr.ImageIdentifier
	for _, img := range images {
		d := img.Digest
		ids = append(ids, &ecr.ImageIdentifier{ImageDigest: &d})
	}
	output, err := s.ecr.BatchDeleteImage(
		&ecr.BatchDeleteImageInput{
			ImageIds:       ids,
			RepositoryName: &repo,
		},
	)
	if err != nil {
		return nil, err
	}
	for _, awsImgId := range output.ImageIds {
		imgId := model.ImageId{Digest: *awsImgId.ImageDigest, Tag: *awsImgId.ImageTag}
		result.Deletions = append(result.Deletions, imgId)
	}
	for _, awsFailure := range output.Failures {
		awsImgId := *awsFailure.ImageId
		imgId := model.ImageId{Digest: *awsImgId.ImageDigest, Tag: *awsImgId.ImageTag}
		failure := model.ImageFailure{Id: imgId, Code: *awsFailure.FailureCode, Reason: *awsFailure.FailureReason}
		result.Failures = append(result.Failures, failure)
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
