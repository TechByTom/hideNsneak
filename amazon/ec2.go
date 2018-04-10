package amazon

import (
	"log"
	"os"
	"path"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type EC2Config struct {
	Count   int
	ImageID string
	Region  string
}

//Deploy multiple EC2 instances across regions and return Instance
func DeployInstances(secret string, accessID string, regionList []string, imageIDList []string, number int, publicKey string, instanceType string) []*ec2.Instance {
	svc := createEC2Session(regionList[0], secret, accessID)
	describedRegions, err := svc.DescribeRegions(&ec2.DescribeRegionsInput{
		RegionNames: aws.StringSlice(regionList),
	})
	if err != nil {
		log.Println("Unable to describe AWS regions", err)
		os.Exit(1)
	}
	ec2Configs := ec2RegionMap(describedRegions.Regions, number, imageIDList)
	var ec2Instances []*ec2.Instance
	for _, ec2 := range ec2Configs {
		if ec2.Count > 0 {
			tempInstances := deployRegionEC2(ec2.ImageID, int64(ec2.Count), ec2.Region, secret, accessID, publicKey, instanceType)
			if tempInstances == nil {
				log.Println("Error creating instances for region: " + ec2.Region)
			} else {
				ec2Instances = append(ec2Instances, tempInstances...)
			}
		}
	}

	return ec2Instances
}

//Terminate EC2 Instances
func DestroyInstance(region string, instanceId string, secret string, accessID string) error {
	svc := createEC2Session(region, secret, accessID)
	_, err := svc.TerminateInstances(&ec2.TerminateInstancesInput{
		InstanceIds: aws.StringSlice([]string{instanceId}),
	})
	if err != nil {
		log.Println("There was an errror terminating your EC2 instances, go clean it up %s", err)
		return err
	}
	return nil
}

func StartInstance(region string, instanceId string, secret string, accessID string) error {
	svc := createEC2Session(region, secret, accessID)
	_, err := svc.StartInstances(&ec2.StartInstancesInput{
		InstanceIds: aws.StringSlice([]string{instanceId}),
	})
	if err != nil {
		return err
	}
	return nil
}

func StopInstance(region string, instanceId string, secret string, accessID string) error {
	svc := createEC2Session(region, secret, accessID)
	_, err := svc.StopInstances(&ec2.StopInstancesInput{
		InstanceIds: aws.StringSlice([]string{instanceId}),
	})
	if err != nil {
		return err
	}
	return nil
}

//Deploy EC2 images by region
func deployRegionEC2(imageID string, count int64, region string, secret string, accessID string, publicKey string, instanceType string) []*ec2.Instance {
	securityGroup := [...]string{"default"}
	svc := createEC2Session(region, secret, accessID)

	keyName := importEC2Key(publicKey, svc)

	//Create Instance
	runResult, err := svc.RunInstances(&ec2.RunInstancesInput{
		ImageId:          aws.String(imageID),
		InstanceType:     aws.String(instanceType),
		SecurityGroupIds: aws.StringSlice(securityGroup[0:]),
		MinCount:         aws.Int64(count),
		MaxCount:         aws.Int64(count),
		KeyName:          aws.String(keyName),
	})
	if err != nil {
		log.Println("Problem Creating Instance: ", err)
		log.Println("Terminating Instances")
		return nil
	}

	return runResult.Instances
}

//Map regions to imageIDs
func ec2RegionMap(regionList []*ec2.Region, count int, imageIDList []string) []EC2Config {
	if len(regionList) > len(imageIDList) {
		log.Println("You have more regions than instances")
		return nil
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
	return ec2Configs
}

////////////////////
////MISC Methods////
////////////////////

func GetEC2IP(region string, secret string, accessID string, instanceId string) string {
	svc := createEC2Session(region, secret, accessID)

	result, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: aws.StringSlice([]string{instanceId}),
	})
	if err != nil {
		log.Printf("Error describing instances while fetching IP address: %s", err)
	}
	return aws.StringValue(result.Reservations[0].Instances[0].PublicIpAddress)
}

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
