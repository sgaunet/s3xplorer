package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

// func printID(cfg aws.Config) error {
// 	client := sts.NewFromConfig(cfg)
// 	identity, err := client.GetCallerIdentity(
// 		context.TODO(),
// 		&sts.GetCallerIdentityInput{},
// 	)
// 	if err != nil {
// 		return err
// 	}
// 	fmt.Printf(
// 		"Account: %s\nUserID: %s\nARN: %s\n\n",
// 		aws.ToString(identity.Account),
// 		aws.ToString(identity.UserId),
// 		aws.ToString(identity.Arn),
// 	)
// 	return err
// }

func (s *App) GetAwsConfig() (cfg aws.Config, err error) {
	if s.cfg.S3endpoint != "" {
		staticResolver := aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
			return aws.Endpoint{
				PartitionID:       "aws",
				URL:               s.cfg.S3endpoint, // or where ever you ran minio
				SigningRegion:     s.cfg.S3Region,
				HostnameImmutable: true,
			}, nil
		})

		cfg = aws.Config{
			Region:           s.cfg.S3Region,
			Credentials:      credentials.NewStaticCredentialsProvider(s.cfg.S3ApikKey, s.cfg.S3accessKey, ""),
			EndpointResolver: staticResolver,
		}
		return
	}

	if s.cfg.SsoAwsProfile != "" {
		fmt.Println("Try to use SSO profile")
		cfg, err = config.LoadDefaultConfig(
			context.TODO(),
			config.WithSharedConfigProfile(s.cfg.SsoAwsProfile),
		)
		return
	}

	if s.cfg.S3ApikKey == "" && s.cfg.S3accessKey == "" {
		cfg, err = config.LoadDefaultConfig(context.TODO(), config.WithRegion(s.cfg.S3Region))
		return
	}
	return cfg, errors.New("no method to initialize aws.Config")
}
