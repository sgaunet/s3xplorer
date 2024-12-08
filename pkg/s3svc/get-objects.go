package s3svc

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/sgaunet/s3xplorer/pkg/dto"
)

// GetObjects returns a list of objects in the parentFolder
func (s *Service) GetObjects(parentFolder string) (result []dto.S3Object, err error) {
	var prefix string = parentFolder
	var delimeter string = "/"

	paginator := s3.NewListObjectsV2Paginator(s.awsS3Client, &s3.ListObjectsV2Input{
		Bucket:    aws.String(s.cfg.Bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String(delimeter),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		for _, obj := range page.Contents {
			isDownloadable, isRestoring, err := s.IsDownloadable(*obj.Key)
			if err != nil {
				return nil, err
			}
			result = append(result, dto.S3Object{
				Key:            *obj.Key,
				Size:           *obj.Size,
				LastModified:   *obj.LastModified,
				ETag:           *obj.ETag,
				StorageClass:   string(obj.StorageClass),
				IsDownloadable: isDownloadable,
				IsRestoring:    isRestoring,
			})
		}
	}
	return result, nil
}

// SearchObjects returns a list of objects in the parentFolder that match the fileToSearch
func (s *Service) SearchObjects(prefix string, fileToSearch string) (result []dto.S3Object, err error) {
	var delimeter string = "/"

	if fileToSearch == "" {
		return nil, nil
	}

	folders, err := s.GetAllFolders(prefix)
	if err != nil {
		return nil, err
	}

	for _, folder := range folders {
		paginator := s3.NewListObjectsV2Paginator(s.awsS3Client, &s3.ListObjectsV2Input{
			Bucket:    aws.String(s.cfg.Bucket),
			Prefix:    aws.String(folder.Key),
			Delimiter: aws.String(delimeter),
		})

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(context.TODO())
			if err != nil {
				return nil, err
			}
			for _, obj := range page.Contents {
				if strings.Contains(*obj.Key, fileToSearch) {
					isDownloadable, isRestoring, err := s.IsDownloadable(*obj.Key)
					if err != nil {
						return nil, err
					}
					result = append(result, dto.S3Object{
						Key:            *obj.Key,
						Size:           *obj.Size,
						LastModified:   *obj.LastModified,
						ETag:           *obj.ETag,
						StorageClass:   string(obj.StorageClass),
						IsDownloadable: isDownloadable,
						IsRestoring:    isRestoring,
					})
				}
			}
		}
	}
	return result, nil
}
