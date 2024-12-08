package s3svc

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/sgaunet/s3xplorer/pkg/dto"
)

// GetFolders returns a list of folders in the parentFolder
func (s *Service) GetFolders(parentFolder string) (result []dto.S3Object, err error) {
	var delimeter string = "/"

	paginator := s3.NewListObjectsV2Paginator(s.awsS3Client, &s3.ListObjectsV2Input{
		Bucket:    aws.String(s.cfg.Bucket),
		Prefix:    aws.String(parentFolder),
		Delimiter: aws.String(delimeter),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		for _, prefix := range page.CommonPrefixes {
			obj := dto.S3Object{
				Key:            *prefix.Prefix,
				Size:           0,
				LastModified:   time.Time{},
				ETag:           "",
				StorageClass:   "",
				IsDownloadable: false,
				IsRestoring:    false,
			}
			result = append(result, obj)
		}
	}
	return result, nil
}

// GetAllFolders returns a list of all folders in the parentFolder and its subfolders
func (s *Service) GetAllFolders(parentFolder string) (result []dto.S3Object, err error) {
	folders, err := s.GetFolders(parentFolder)
	if err != nil {
		return nil, err
	}
	if len(folders) == 0 {
		return nil, nil
	}

	for _, folder := range folders {
		result = append(result, folder)
		subFolders, err := s.GetAllFolders(folder.Key)
		if err != nil {
			return nil, err
		}
		if len(subFolders) == 0 {
			continue
		}
		result = append(result, subFolders...)
	}
	return result, nil
}
