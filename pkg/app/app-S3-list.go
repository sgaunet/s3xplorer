package app

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
