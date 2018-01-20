package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type EC2Config struct {
	Count   int
	ImageID string
	Region  string
}

////////////////////
////MISC Methods////
////////////////////

func createEC2Session(region string, secret string, accessID string) *ec2.EC2 {
	//Set Session
	svc := ec2.New(session.New(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessID, secret, ""),
	}))
	return svc
}

//TODO: Update this so it checks to see if the public key
//has a name on EC2. If not, then import it and return the name.
func importEC2Key(pubkey string, svc *ec2.EC2) string {
	return path.Base(pubkey)
}

func getEC2IP(region string, secret string, accessID string, instanceId string) string {
	svc := createEC2Session(region, secret, accessID)
	result, _ := svc.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: aws.StringSlice([]string{instanceId}),
	})
	return aws.StringValue(result.Reservations[0].Instances[0].PublicIpAddress)
}

//Terminate EC2 Instances
func terminateEC2Instances(regionIDMap map[string][]string, secret string, accessID string) {
	for region := range regionIDMap {
		if len(regionIDMap[region]) > 0 {
			svc := createEC2Session(region, secret, accessID)
			_, err := svc.TerminateInstances(&ec2.TerminateInstancesInput{
				InstanceIds: aws.StringSlice(regionIDMap[region]),
			})
			if err != nil {
				log.Println("There was an errror terminating your EC2 instances, go clean it up %s", err)
			} else {
				log.Println("Successfully deleted " + region + "instances")
			}
		}
	}
}

//Deploy EC2 images by region
func deployRegionEC2(imageID string, count int64, region string, config Config) ([]CloudInstance, int) {
	securityGroup := [...]string{"default"}
	var ec2CloudInstances []CloudInstance
	svc := createEC2Session(region, config.AWS.Secret, config.AWS.AccessID)

	keyName := importEC2Key(config.PublicKey, svc)

	//Create Instance
	runResult, err := svc.RunInstances(&ec2.RunInstancesInput{
		ImageId:          aws.String(imageID),
		InstanceType:     aws.String(config.AWS.Type),
		SecurityGroupIds: aws.StringSlice(securityGroup[0:]),
		MinCount:         aws.Int64(count),
		MaxCount:         aws.Int64(count),
		KeyName:          aws.String(keyName),
	})
	if err != nil {
		log.Println("Problem Creating Instance: ", err)
		log.Println("Terminating Instances")
		return ec2CloudInstances, 1
	}

	var ec2CloudInstance CloudInstance
	privKey := strings.Split(config.PublicKey, ".")
	for _, instance := range runResult.Instances {
		ec2Cloudinstance.Cloud.ID = aws.StringValue(instance.InstanceId)
		ec2Cloudinstance.Cloud.IPv4 = aws.StringValue(instance.PublicIpAddress)
		ec2Cloudinstance.Cloud.Type = "EC2"
		ec2Cloudinstance.SSH.Username = "ubuntu"
		ec2Cloudinstance.Proxy.SOCKSActive = false
		ec2CloudInstance.SOCKSPort = "0"
		ec2Cloudinstance.Cloud.Region = region
		ec2Cloudinstance.SSH.PrivateKey = strings.Join(privKey[:len(privKey)-1], ".")
		ec2Cloudinstance.Cloud.IPv4 = config
		ec2CloudInstances = append(ec2CloudInstances, ec2CloudInstance)
	}
	return ec2CloudInstances, 0
}

//Map regions to imageIDs
func ec2RegionMap(regionList []*ec2.Region, count int, imageIDList []string) []EC2Config {
	finalConfigs := make([]EC2Config, 0, 100)
	if len(regionList) != len(imageIDList) {
		log.Println("You don't have the same number of regions and ami IDs")
		os.Exit(1)
	}
	ec2Configs := make([]EC2Config, len(regionList))
	countPerRegion := count / len(regionList)
	countRemainder := count % len(regionList)
	for c, p := range regionList {
		ec2Configs[c] = EC2Config{
			Count:   countPerRegion,
			ImageID: imageIDList[c],
			Region:  *p.RegionName,
		}
	}
	counter := 0
	for c := range ec2Configs {
		if counter < countRemainder {
			ec2Configs[c].Count = ec2Configs[c].Count + 1
			counter = counter + 1
		} else {
			break
		}
	}
	for _, p := range ec2Configs {
		if p.Count != 0 {
			finalConfigs = append(finalConfigs, p)
		}
	}
	return finalConfigs
}

//Deploy multiple EC2 instances across regions and return cloudInstance
func deployMultipleEC2(config Config) ([]CloudInstance, int, map[string][]string) {
	var tempCloudInstances []CloudInstance
	var intResult int
	var errorResult int
	terminationMap := make(map[string][]string)
	regionList := strings.Split(config.AWS.Regions, ",")
	imageIDList := strings.Split(config.AWS.ImageIDs, ",")

	svc := createEC2Session(regionList[0], config.AWS.Secret, config.AWS.AccessID)
	describedRegions, err := svc.DescribeRegions(&ec2.DescribeRegionsInput{
		RegionNames: aws.StringSlice(regionList),
	})
	if err != nil {
		log.Println("Unable to describe AWS regions", err)
		os.Exit(1)
	}
	ec2Map := ec2RegionMap(describedRegions.Regions, config.AWS.Number, imageIDList)
	var ec2CloudInstances []CloudInstance
	for _, ec2 := range ec2Map {
		tempCloudInstances, intResult = deployRegionEC2(ec2.ImageID, int64(ec2.Count), ec2.Region, config)
		if intResult == 1 {
			errorResult = intResult
		}
		ec2CloudInstances = append(ec2CloudInstances, tempCloudInstances...)
	}
	var tempArray []string
	for _, region := range regionList {
		for _, instance := range ec2CloudInstances {
			if instance.Cloud.Region == region {
				tempArray = append(tempArray, instance.Cloud.ID)
			}
		}
		terminationMap[region] = tempArray
		tempArray = []string{}
	}
	return ec2CloudInstances, errorResult, terminationMap
}

//List EC2 Instances

/////////////////////
///Security Groups///
/////////////////////

//TODO: Possibly add config variable setting the DescribeSecurityGroupsOutput for later use

//TODO: Add ability to update current configs with existing security groups
func describeRegionSecurityGroup(config *Config, region string) *ec2.DescribeSecurityGroupsOutput {
	svc := createEC2Session(region, config.AWS.Secret, config.AWS.AccessID)
	securityGroups, err := svc.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		GroupIds: aws.StringSlice(config.AWS.SecurityGroups[region]),
	})
	if err != nil {
		log.Println("Error describing security group for region: " + region)
	}
	return securityGroups
}

func (instance *CloudInstance) describeInstanceSecurityGroup() *ec2.DescribeSecurityGroupsOutput {
	svc := createEC2Session(instance.Cloud.Region, instance.Cloud.IPv4.AWS.Secret, instance.Cloud.IPv4.AWS.AccessID)
	securityGroups, err := svc.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		GroupIds: aws.StringSlice(instance.Cloud.Firewalls),
	})
	if err != nil {
		log.Println("Error describing security group for AWS Instance: " + instance.Cloud.ID)
	}
	return securityGroups
}

func describeAllSecurityGroups(config *Config) {
	for region := range config.AWS.SecurityGroups {
		describeRegionSecurityGroup(config, region)
	}
}

//Create Security Group
//TODO: Allow option for UDP port speicification
func createSecurityGroup(securityGroup string, desc string, ipPortMap map[string][]int, region string, config *Config) string {
	svc := createEC2Session(region, config.AWS.Secret, config.AWS.AccessID)
	createRes, err := svc.CreateSecurityGroup(&ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(securityGroup),
		Description: aws.String(desc),
	})
	if val, ok := config.AWS.SecurityGroups[region]; ok {
		config.AWS.SecurityGroups[region] = append(val, *createRes.GroupId)
	} else {
		config.AWS.SecurityGroups[region] = strings.Split(*createRes.GroupId, "AHUIGFDKJS")
	}
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "InvalidGroup.Duplicate":
				log.Printf("Security group %q already exists.", securityGroup)
			}
		}
		log.Printf("Unable to create security group %q, %v", securityGroup, err)
	}

	//TODO: implement more robust check to see if they want to restrict ssh to their own IP, note
	//that this could cause issues connecting and would require them to update the security groups via the console
	//especially if they have a natted IP address
	ec2Permissions := []*ec2.IpPermission{
		(&ec2.IpPermission{}).
			SetIpProtocol("tcp").
			SetFromPort(int64(22)).
			SetToPort(int64(22)).
			SetIpRanges([]*ec2.IpRange{
				{CidrIp: aws.String("0.0.0.0/0")},
			}),
	}

	if len(ipPortMap) > 0 {
		for ip, ports := range ipPortMap {
			for port := range ports {
				ec2Permissions = append(ec2Permissions, (&ec2.IpPermission{}).
					SetIpProtocol("tcp").
					SetFromPort(int64(port)).
					SetToPort(int64(port)).
					SetIpRanges([]*ec2.IpRange{
						{CidrIp: aws.String(ip)},
					}),
				)
			}
		}
	}
	// fmt.Printf("Created security group %s with VPC %s.\n", aws.StringValue(createRes.GroupId), instance.Cloud.ID)
	_, err = svc.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
		GroupName: aws.String(securityGroup),
		IpPermissions: []*ec2.IpPermission{
			(&ec2.IpPermission{}).
				SetIpProtocol("tcp").
				SetFromPort(22).
				SetToPort(22).
				SetIpRanges([]*ec2.IpRange{
					(&ec2.IpRange{}).
						SetCidrIp("0.0.0.0/0"),
				}),
		},
	})
	if err != nil {
		log.Printf("Unable to set security group %q ingress, %v", securityGroup, err)
	}
	fmt.Println("Successfully set security group ingress")
	return *createRes.GroupId
}

//TODO: Delete security group from instances

//Delete Security Group
func deleteSecurityGroup(region string, groupID string, config *Config) {
	svc := createEC2Session(region, config.AWS.Secret, config.AWS.AccessID)
	_, err := svc.DeleteSecurityGroup(&ec2.DeleteSecurityGroupInput{
		GroupId: aws.String(groupID),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "InvalidGroupId.Malformed":
				fallthrough
			case "InvalidGroup.NotFound":
				log.Printf("%s.", aerr.Message())
			}
		}
		log.Printf("Unable to get descriptions for security groups, %v.", err)
	}
	log.Printf("Successfully delete security group %q.\n", groupID)
}

func (instance *CloudInstance) setSecurityGroup(securityGroups []string) {
	svc := createEC2Session(instance.Cloud.Region, instance.Cloud.IPv4.AWS.Secret, instance.Cloud.IPv4.AWS.AccessID)
	_, err := svc.ModifyInstanceAttribute(&ec2.ModifyInstanceAttributeInput{
		InstanceId: aws.String(instance.Cloud.ID),
		Groups:     aws.StringSlice(securityGroups),
	})
	if err != nil {
		log.Println("Error editing security group for instance")
	} else {
		instance.Cloud.Firewalls = securityGroups
		log.Println("Successfully edited security group for instance")
	}
}

//TODO: Add functionality to edit rules of existing security group. For now we will just create/delete upon the
//creation/deletion of rule sets

//////////////////////
/////CloudFronting////
//////////////////////

func createCloudFront(config Config, comment string, domainName string) *cloudfront.Distribution {
	originId := (config.Customer + "-" + domainName)
	svc := cloudfront.New(session.New(&aws.Config{
		Credentials: credentials.NewStaticCredentials(config.AWS.AccessID, config.AWS.Secret, ""),
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
				TargetOriginId:  aws.String(originId),
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
						Id:         aws.String(originId),
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

func disableCloudFront(distribution *cloudfront.Distribution, ETag string, config Config) (string, string) {
	svc := cloudfront.New(session.New(&aws.Config{
		Credentials: credentials.NewStaticCredentials(config.AWS.AccessID, config.AWS.Secret, ""),
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

func deleteCloudFront(id string, ETag string, config Config) {
	svc := cloudfront.New(session.New(&aws.Config{
		Credentials: credentials.NewStaticCredentials(config.AWS.AccessID, config.AWS.Secret, ""),
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

func listCloudFront(config Config) []*cloudfront.DistributionSummary {
	svc := cloudfront.New(session.New(&aws.Config{
		Credentials: credentials.NewStaticCredentials(config.AWS.AccessID, config.AWS.Secret, ""),
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

func getCloudFront(distributionID string, config Config) (*cloudfront.Distribution, string) {
	svc := cloudfront.New(session.New(&aws.Config{
		Credentials: credentials.NewStaticCredentials(config.AWS.AccessID, config.AWS.Secret, ""),
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
