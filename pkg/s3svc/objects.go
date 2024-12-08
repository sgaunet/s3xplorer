package s3svc

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

const DefaultRetentionPolicyInDays int32 = 2

// IsDownloadable returns true if the object is downloadable
func (s *Service) IsDownloadable(key string) (isDownloadable bool, soon bool, err error) {

	hi := s3.HeadObjectInput{
		Bucket: &s.cfg.Bucket,
		Key:    &key,
	}
	o, err := s.awsS3Client.HeadObject(context.TODO(), &hi)
	if err != nil {
		return
	}

	// fmt.Printf("%+v\n", o)
	// fmt.Printf("%+v\n", *o.Restore)
	// fmt.Printf("%+v\n", o.StorageClass)
	if o.StorageClass == "" || o.StorageClass == "STANDARD" {
		return true, false, err
	}

	if o.Restore != nil {
		res := conv(strings.ReplaceAll(*o.Restore, ", ", " "))
		if vv, ok := res["ongoing-request"]; ok {
			// fmt.Println(vv)
			if vv == "\"false\"" {
				soon = false
			}
			if vv == "\"true\"" {
				soon = true
				return
			}
		}
		if vv, ok := res["expiry-date"]; ok {
			// Declaring layout constant
			const layout = "\"Mon 2 Jan 2006 15:04:06 MST\""
			// Calling Parse() method with its parameters
			tm, err2 := time.Parse(layout, vv)
			if err2 != nil {
				s.log.Errorln("problem when parsing time:", err2.Error())
			}
			// Returns output
			if time.Now().After(tm) {
				isDownloadable = true
				return
			}
		}
	}
	return
}

// RestoreObject restores an object
func (s *Service) RestoreObject(key string) (err error) {
	// https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/s3@v1.26.0/types#RestoreRequest
	tt := types.GlacierJobParameters{
		Tier: "Standard",
	}
	r := types.RestoreRequest{
		Days: aws.Int32(DefaultRetentionPolicyInDays),
		// Type:           "SELECT",
		GlacierJobParameters: &tt,
		// Tier: "Standard",
		// OutputLocation: &x,
		// Description:    &i,
	}
	p := s3.RestoreObjectInput{
		Bucket:         &s.cfg.Bucket,
		Key:            &key,
		RestoreRequest: &r,
	}
	o, err := s.awsS3Client.RestoreObject(context.TODO(), &p)
	if err != nil {
		return err
	}
	s.log.Debugf("%v", o)
	return
}

func (s *Service) HeadObject(key string) (err error) {
	hi := s3.HeadObjectInput{
		Bucket: &s.cfg.Bucket,
		Key:    &key,
	}
	o, err := s.awsS3Client.HeadObject(context.TODO(), &hi)
	if err != nil {
		return err
	}
	// fmt.Printf("%+v\n", o)
	// fmt.Printf("%+v\n", *o.Restore)
	// fmt.Printf("%+v\n", o.StorageClass)

	res := conv(strings.ReplaceAll(*o.Restore, ", ", " "))
	if vv, ok := res["ongoing-request"]; ok {
		fmt.Println(vv)
	}
	if vv, ok := res["expiry-date"]; ok {
		fmt.Println(vv)
	}

	return
}

func conv(str string) map[string]string {
	lastQuote := rune(0)
	f := func(c rune) bool {
		switch {
		case c == lastQuote:
			lastQuote = rune(0)
			return false
		case lastQuote != rune(0):
			return false
		case unicode.In(c, unicode.Quotation_Mark):
			lastQuote = c
			return false
		default:
			return unicode.IsSpace(c)

		}
	}

	// splitting string by space but considering quoted section
	items := strings.FieldsFunc(str, f)

	// create and fill the map
	m := make(map[string]string)
	for _, item := range items {
		x := strings.Split(item, "=")
		m[x[0]] = x[1]
	}

	// print the map
	for k, v := range m {
		fmt.Printf("%s: %s\n", k, v)
	}
	return m
}
