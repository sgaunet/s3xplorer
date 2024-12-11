package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

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

// GetAwsConfig returns an aws.Config
func (s *App) GetAwsConfig() (cfg aws.Config, err error) {
	if s.cfg.S3endpoint != "" {
		s.log.Debug("Try to use S3 endpoint")
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
		s.log.Debug("Try to use SSO profile")
		cfg, err = config.LoadDefaultConfig(
			context.TODO(),
			config.WithSharedConfigProfile(s.cfg.SsoAwsProfile),
		)
		if err != nil {
			s.log.Error("Error loading SSO profile", slog.String("error", err.Error()))
			return cfg, fmt.Errorf("error loading SSO profile: %w", err)
		}
		s.log.Debug("SSO profile loaded")
		return cfg, nil
	}

	if s.cfg.S3ApikKey == "" && s.cfg.S3accessKey == "" {
		cfg, err = config.LoadDefaultConfig(context.TODO(), config.WithRegion(s.cfg.S3Region))
		if err != nil {
			s.log.Error("Error loading default config", slog.String("error", err.Error()))
			return cfg, fmt.Errorf("error loading default config: %w", err)
		}
		s.log.Debug("Default config loaded")
		return cfg, nil
	}
	return cfg, errors.New("no method to initialize aws.Config")
}
