// Package dto provides data transfer objects for S3 operations
package dto

import "time"

// S3Object is the structure to store the S3 object metadata.
type S3Object struct {
	ETag           string    `json:"etag"`
	Key            string    `json:"key"`
	Name           string    `json:"name"`
	LastModified   time.Time `json:"lastmodified"`
	Size           int64     `json:"size"`
	SizeHuman      string    `json:"sizeHuman"`
	StorageClass   string    `json:"storageclass"`
	IsFolder       bool      `json:"isFolder"`
	Prefix         string    `json:"prefix"`
	IsDownloadable bool
	IsRestoring    bool
}

// Bucket represents an S3 bucket.
type Bucket struct {
	Name         string    `json:"name"`
	Region       string    `json:"region"`
	CreationDate time.Time `json:"creationDate"`
}
