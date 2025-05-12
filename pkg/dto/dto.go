// Package dto provides data transfer objects for S3 operations
package dto

import "time"

// S3Object is the structure to store the S3 object metadata.
type S3Object struct {
	ETag           string    `json:"etag"`
	Key            string    `json:"key"`
	LastModified   time.Time `json:"lastmodified"`
	Size           int64     `json:"size"`
	StorageClass   string    `json:"storageclass"`
	IsDownloadable bool
	IsRestoring    bool
}

// Bucket represents an S3 bucket.
type Bucket struct {
	Name         string    `json:"name"`
	CreationDate time.Time `json:"creationDate"`
}
