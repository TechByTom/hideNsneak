package amazon

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudfront"
)

/////CloudFronting////
//////////////////////

func createCloudFront(client string, comment string, domainName string, secret string, accessID string) *cloudfront.Distribution {
	originID := (client + "-" + domainName)
	svc := cloudfront.New(session.New(&aws.Config{
		Credentials: credentials.NewStaticCredentials(accessID, secret, ""),
	}))
	// sslProtos := &cloudfront.OriginSslProtocols{
	// 	Items: aws.StringSlice([]string{"SSLv3","TLSv1","TLSv1.1","TLS1.2"}),
	// 	Quantity: aws.Int64(int64(4)),
	// }

	// err := sslProtos.Validate()
	// if err != nil {
	// 	fmt.Println("Problem validating ssl protocols %s", err)
	// 	return
	// }

	distributionOutput, err := svc.CreateDistribution(&cloudfront.CreateDistributionInput{
		DistributionConfig: &cloudfront.DistributionConfig{
			Aliases: &cloudfront.Aliases{
				Quantity: aws.Int64(int64(0)),
			},
			CacheBehaviors: &cloudfront.CacheBehaviors{
				Quantity: aws.Int64(int64(0)),
			},
			CallerReference: aws.String(time.Now().String()),
			Comment:         aws.String(comment),
			CustomErrorResponses: &cloudfront.CustomErrorResponses{
				Quantity: aws.Int64(int64(0)),
			},
			DefaultCacheBehavior: &cloudfront.DefaultCacheBehavior{
				AllowedMethods: &cloudfront.AllowedMethods{
					CachedMethods: &cloudfront.CachedMethods{
						Items:    aws.StringSlice([]string{"HEAD", "GET"}),
						Quantity: aws.Int64(int64(2)),
					},
					Items:    aws.StringSlice([]string{"HEAD", "DELETE", "POST", "GET", "OPTIONS", "PUT", "PATCH"}),
					Quantity: aws.Int64(int64(7)),
				},
				Compress:   aws.Bool(false),
				DefaultTTL: aws.Int64(int64(0)),
				ForwardedValues: &cloudfront.ForwardedValues{
					Cookies: &cloudfront.CookiePreference{
						Forward: aws.String("all"),
					},
					Headers: &cloudfront.Headers{
						Quantity: aws.Int64(int64(0)),
					},
					QueryString: aws.Bool(true),
					QueryStringCacheKeys: &cloudfront.QueryStringCacheKeys{
						Quantity: aws.Int64(int64(0)),
					},
				},
				LambdaFunctionAssociations: &cloudfront.LambdaFunctionAssociations{
					Quantity: aws.Int64(int64(0)),
				},
				MaxTTL:          aws.Int64(int64(0)),
				MinTTL:          aws.Int64(int64(0)),
				SmoothStreaming: aws.Bool(false),
				TargetOriginId:  aws.String(originID),
				TrustedSigners: &cloudfront.TrustedSigners{
					Enabled:  aws.Bool(false),
					Quantity: aws.Int64(int64(0)),
				},
				ViewerProtocolPolicy: aws.String("allow-all"),
			},
			Enabled:       aws.Bool(true),
			IsIPV6Enabled: aws.Bool(false),
			Origins: &cloudfront.Origins{
				Items: []*cloudfront.Origin{
					&cloudfront.Origin{
						CustomHeaders: &cloudfront.CustomHeaders{
							Quantity: aws.Int64(int64(0)),
						},
						CustomOriginConfig: &cloudfront.CustomOriginConfig{
							HTTPPort:               aws.Int64(int64(80)),
							HTTPSPort:              aws.Int64(int64(443)),
							OriginKeepaliveTimeout: aws.Int64(int64(5)),
							OriginProtocolPolicy:   aws.String("match-viewer"),
							OriginReadTimeout:      aws.Int64(int64(30)),
							OriginSslProtocols: &cloudfront.OriginSslProtocols{
								Items:    aws.StringSlice([]string{"TLSv1", "TLSv1.1", "TLSv1.2"}),
								Quantity: aws.Int64(int64(3))},
						},
						DomainName: aws.String(domainName),
						Id:         aws.String(originID),
						OriginPath: aws.String(""),
					},
				},
				Quantity: aws.Int64(int64(1)),
			},
			PriceClass: aws.String("PriceClass_All"),
			Restrictions: &cloudfront.Restrictions{
				GeoRestriction: &cloudfront.GeoRestriction{
					Quantity:        aws.Int64(int64(0)),
					RestrictionType: aws.String("none"),
				},
			},
		},
	})

	//Dig further into AWS cloudfront error codes and add a switch statement to catch them
	if err != nil {
		fmt.Printf("There was a problem creating the cloudfront distribution: %s", err)
		return nil
	}
	fmt.Println("Distribution Created")
	fmt.Println(*distributionOutput.Distribution)

	fmt.Println("Cloudfront URL")
	fmt.Println(*distributionOutput.Location)

	return distributionOutput.Distribution
}

func disableCloudFront(distribution *cloudfront.Distribution, ETag string, secret string, accessID string) (string, string) {
	svc := cloudfront.New(session.New(&aws.Config{
		Credentials: credentials.NewStaticCredentials(accessID, secret, ""),
	}))
	newDistributionConfig := distribution.DistributionConfig
	newDistributionConfig.SetEnabled(false)
	disableDistributionOutput, err := svc.UpdateDistribution(&cloudfront.UpdateDistributionInput{
		Id:                 distribution.Id,
		DistributionConfig: newDistributionConfig,
		IfMatch:            aws.String(ETag),
	})
	fmt.Println(newDistributionConfig)
	//Add AWS Custom error codes into here
	if err != nil {
		fmt.Printf("Problem disabling cloudfront distribution: %s", err)
		return "", ""
	}
	return *disableDistributionOutput.Distribution.Id, *disableDistributionOutput.ETag
}

func deleteCloudFront(id string, ETag string, secret string, accessID string) {
	svc := cloudfront.New(session.New(&aws.Config{
		Credentials: credentials.NewStaticCredentials(accessID, secret, ""),
	}))
	_, err := svc.DeleteDistribution(&cloudfront.DeleteDistributionInput{
		Id:      aws.String(id),
		IfMatch: aws.String(ETag),
	})
	if err != nil {
		fmt.Printf("Error deleting instance, instance is now disabled: %s", err)
		return
	}
}

func listCloudFront(accessID string, secret string) []*cloudfront.DistributionSummary {
	svc := cloudfront.New(session.New(&aws.Config{
		Credentials: credentials.NewStaticCredentials(accessID, secret, ""),
	}))
	distributionList, err := svc.ListDistributions(&cloudfront.ListDistributionsInput{
		MaxItems: aws.Int64(int64(10)),
	})
	if err != nil {
		fmt.Println("Error returning cloudfront object")
	}
	fmt.Println(distributionList.DistributionList.Items[0])
	return distributionList.DistributionList.Items
}

func getCloudFront(distributionID string, accessID string, secret string) (*cloudfront.Distribution, string) {
	svc := cloudfront.New(session.New(&aws.Config{
		Credentials: credentials.NewStaticCredentials(accessID, secret, ""),
	}))
	distributionOutput, err := svc.GetDistribution(&cloudfront.GetDistributionInput{
		Id: aws.String(distributionID),
	})
	if err != nil {
		fmt.Println("Error getting cloudfront object")
		return nil, ""
	}
	return distributionOutput.Distribution, *distributionOutput.ETag
}
