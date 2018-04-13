package amazon

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
)

/////////////////////
///Security Groups///
/////////////////////

//TODO: Possibly add config variable setting the DescribeSecurityGroupsOutput for later use

//TODO: Add ability to update current configs with existing security groups
func describeRegionSecurityGroup(secret string, accessID string, region string, securityGroupSlice []string) *ec2.DescribeSecurityGroupsOutput {
	svc := createEC2Session(region, secret, accessID)
	securityGroups, err := svc.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		GroupIds: aws.StringSlice(securityGroupSlice),
	})
	if err != nil {
		log.Println("Error describing security group for region: " + region)
	}
	return securityGroups
}

//TDOD: Get security groups for instance
func describeInstanceSecurityGroup(region string, ip string, secret string, accessID string, securityGroupSlice []string) *ec2.DescribeSecurityGroupsOutput {
	svc := createEC2Session(region, secret, accessID)
	securityGroups, err := svc.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		GroupIds: aws.StringSlice(securityGroupSlice),
	})
	if err != nil {
		log.Println("Error describing security group for AWS Instance")
	}
	return securityGroups
}

//Create Security Group
//TODO: Allow option for UDP port speicification
func CreateSecurityGroup(securityGroup string, desc string, ips []string, ports []int, region string, secret string, accessID string) (string, error) {
	svc := createEC2Session(region, secret, accessID)
	createRes, err := svc.CreateSecurityGroup(&ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(securityGroup),
		Description: aws.String(desc),
	})

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "InvalidGroup.Duplicate":
				log.Printf("Security group %q already exists.", securityGroup)
			}
		}
		log.Printf("Unable to create security group %q, %v", securityGroup, err)
		return "", err
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

	var ipRanges []*ec2.IpRange
	for _, ip := range ips {
		ipRanges = append(ipRanges, &ec2.IpRange{
			CidrIp: aws.String(ip),
		})
	}

	for port := range ports {
		ec2Permissions = append(ec2Permissions, (&ec2.IpPermission{}).
			SetIpProtocol("tcp").
			SetFromPort(int64(port)).
			SetToPort(int64(port)).
			SetIpRanges(ipRanges),
		)
	}
	// fmt.Printf("Created security group %s with VPC %s.\n", aws.StringValue(createRes.GroupId), instance.Cloud.ID)
	_, err = svc.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
		GroupName:     aws.String(securityGroup),
		IpPermissions: ec2Permissions,
	})
	if err != nil {
		log.Printf("Unable to set security group %q ingress, %v", securityGroup, err)
		return "", err
	}
	fmt.Println("Successfully set security group ingress")
	return *createRes.GroupId, nil
}

//TODO: Delete security group from instances

//Delete Security Group
func deleteSecurityGroup(region string, groupID string, secret string, accessID string) {
	svc := createEC2Session(region, secret, accessID)
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

func setSecurityGroup(securityGroups []string, secret string, accessID string, id string, region string) []string {
	emptySlice := securityGroups[:1]
	svc := createEC2Session(region, secret, accessID)
	_, err := svc.ModifyInstanceAttribute(&ec2.ModifyInstanceAttributeInput{
		InstanceId: aws.String(id),
		Groups:     aws.StringSlice(securityGroups),
	})
	if err != nil {
		log.Println("Error editing security group for instance")
		return emptySlice
	}
	log.Println("Successfully edited security group for instance")
	return securityGroups
}

//TODO: Add functionality to edit rules of existing security group. For now we will just create/delete upon the
//creation/deletion of rule sets
