aws-ecr-gc
==========

A garbage collector to delete old Docker images from [Amazon EC2 Container
Registry](https://aws.amazon.com/ecr/) (ECR), which by default has a limit of
1,000 images per repository.

Given a list of tag prefixes, `aws-ecr-gc` deletes all but the newest N images
matching those prefixes. Images with tags not matching the listed prefixes are
not deleted. Optionally, untagged images are also deleted.

AWS authentication via the standard strategies as implemented in
[aws-sdk-go](https://github.com/aws/aws-sdk-go). We recomment
[aws-vault](http://github.com/99designs/aws-vault) to manage these.

Usage
-----

```
Usage of aws-ecr-gc:
  --delete-untagged
        whether to delete untagged images
  --keep value
        map of image tag prefixes to how many to keep, e.g. --keep release=4 --keep build=8
  --region string
        AWS region (defaults to AWS_DEFAULT_REGION from environment)
  -repo string
        AWS ECR repository name
```

Example
-------

From the `testrepo` ECR repository in the `us-east-1` AWS region:

* delete all untagged images,
* delete all but the latest 4 images with tags starting with `release-production`,
* delete all but the latest 8 images with tags starting with `build`.

```
$ export AWS_DEFAULT_REGION=us-east-1
$ aws-ecr-gc --repo testrepo --delete-untagged=true --keep release-production=4 --keep build=8
Total images in testrepo (us-east-1): 47
Images to delete (3)
  2017-03-20 03:51:41: sha256:2a1fce5b2... [build-64cd372]
  2017-03-17 17:12:07: sha256:4fe1451fc... [build-1d293f7]
  2017-03-17 16:58:15: sha256:e0a2a1b4f... [build-6d12484]
Deleted (3)
  sha256:2a1fce5b2... (build-64cd372)
  sha256:4fe1451fc... (build-1d293f7)
  sha256:e0a2a1b4f... (build-6d12484)
Failures (0)
```
