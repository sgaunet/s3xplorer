package s3svc

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/sgaunet/s3xplorer/pkg/dto"
)

// GetObjects returns a list of objects in the parentFolder.
func (s *Service) GetObjects(ctx context.Context, parentFolder string) ([]dto.S3Object, error) {
	// Initialize local result variable
	result := []dto.S3Object{}
	var prefix = parentFolder
	var delimeter = "/"

	paginator := s3.NewListObjectsV2Paginator(s.awsS3Client, &s3.ListObjectsV2Input{
		Bucket:    aws.String(s.cfg.Bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String(delimeter),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("GetObjects: error of paginator.NextPage: %w", err)
		}
		for _, obj := range page.Contents {
			isDownloadable, isRestoring, err := s.IsDownloadable(ctx, *obj.Key)
			if err != nil {
				return nil, fmt.Errorf("GetObjects: error of IsDownloadable: %w", err)
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

// SearchObjects returns a list of objects in the parentFolder that match the fileToSearch.
func (s *Service) SearchObjects(ctx context.Context, prefix string, fileToSearch string) ([]dto.S3Object, error) {
	// Initialize local result variable
	result := []dto.S3Object{}
	var delimeter = "/"
	s.log.Debug("SearchObjects", slog.String("prefix", prefix), slog.String("fileToSearch", fileToSearch))
	if fileToSearch == "" {
		return nil, nil
	}

	folders, err := s.GetAllFolders(ctx, prefix)
	if err != nil {
		return nil, fmt.Errorf("SearchObjects: error of GetAllFolders: %w", err)
	}
	// Add the parent folder to the list of folders
	folders = append(folders, dto.S3Object{Key: prefix})

	for _, folder := range folders {
		paginator := s3.NewListObjectsV2Paginator(s.awsS3Client, &s3.ListObjectsV2Input{
			Bucket:    aws.String(s.cfg.Bucket),
			Prefix:    aws.String(folder.Key),
			Delimiter: aws.String(delimeter),
		})

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, fmt.Errorf("SearchObjects: error of paginator.NextPage: %w", err)
			}
			for _, obj := range page.Contents {
				s.log.Debug("SearchObjects", slog.String("obj.Key", *obj.Key))
				if strings.Contains(*obj.Key, fileToSearch) {
					isDownloadable, isRestoring, err := s.IsDownloadable(ctx, *obj.Key)
					if err != nil {
						return nil, fmt.Errorf("SearchObjects: error of IsDownloadable: %w", err)
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
