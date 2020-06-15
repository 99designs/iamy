package iamy

import (
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
)

var cfnResourceRegexp = regexp.MustCompile(`-[A-Z0-9]{10,20}$`)

type CfnResourceType string

const (
	CfnIamPolicy       = "AWS::IAM::Policy"
	CfnIamRole         = "AWS::IAM::Role"
	CfnIamUser         = "AWS::IAM::User"
	CfnIamGroup        = "AWS::IAM::Group"
	CfnInstanceProfile = "AWS::IAM::InstanceProfile"
	CfnS3Bucket        = "AWS::S3::Bucket"
)

type cfnClient struct {
	cloudformationiface.CloudFormationAPI
	managedResources map[string]CfnResourceTypes
}

func newCfnClient(sess *session.Session) *cfnClient {
	return &cfnClient{
		CloudFormationAPI: cloudformation.New(sess),
	}
}

// PopulateMangedResourceData enumerates all cloudformation stacks and resources to build an internal list of all
// resources that are managed by cloudformation. This list can then be checked by IsManagedResource
func (c *cfnClient) PopulateMangedResourceData() error {
	c.managedResources = map[string]CfnResourceTypes{}
	var nextStack *string

	for {
		stacks, err := c.ListStacks(&cloudformation.ListStacksInput{
			NextToken: nextStack,
			StackStatusFilter: []*string{
				aws.String("CREATE_IN_PROGRESS"),
				aws.String("CREATE_COMPLETE"),
				aws.String("ROLLBACK_COMPLETE"),
				aws.String("IMPORT_COMPLETE"),
				aws.String("REVIEW_IN_PROGRESS"),
				aws.String("CREATE_IN_PROGRESS"),
				aws.String("UPDATE_ROLLBACK_COMPLETE"),
				aws.String("UPDATE_IN_PROGRESS"),
				aws.String("UPDATE_COMPLETE_CLEANUP_IN_PROGRESS"),
				aws.String("UPDATE_COMPLETE"),
				aws.String("UPDATE_ROLLBACK_IN_PROGRESS"),
				aws.String("UPDATE_ROLLBACK_FAILED"),
				aws.String("UPDATE_ROLLBACK_COMPLETE_CLEANUP_IN_PROGRESS"),
				aws.String("UPDATE_ROLLBACK_COMPLETE"),
				aws.String("REVIEW_IN_PROGRESS"),
			},
		})
		if awserr, ok := err.(awserr.Error); ok && awserr != nil && awserr.Code() == "Throttling" {
			time.Sleep(1 * time.Second)
			continue
		}
		if err != nil {
			return err
		}

		for _, stack := range stacks.StackSummaries {
			var nextResource *string
			for {
				resources, err := c.ListStackResources(&cloudformation.ListStackResourcesInput{
					NextToken: nextResource,
					StackName: stack.StackName,
				})
				if awserr, ok := err.(awserr.Error); ok && awserr != nil && awserr.Code() == "Throttling" {
					time.Sleep(1 * time.Second)
					continue
				}
				if err != nil {
					return err
				}

				for _, resource := range resources.StackResourceSummaries {
					if resource.PhysicalResourceId == nil {
						continue
					}
					resType := CfnResourceType(*resource.ResourceType)
					if resType == "AWS::IAM::ManagedPolicy" {
						resType = CfnIamPolicy // we dont care about the distinction as they are both in the "policy" namespace
					}
					name := *resource.PhysicalResourceId
					// Dont know why, but some physical ids are arns, instead of names...
					if strings.HasPrefix(*resource.PhysicalResourceId, "arn:aws:iam") {
						parts := strings.Split(*resource.PhysicalResourceId, "/")
						name = parts[len(parts)-1]
					}

					if !resType.isInterestingResource() {
						continue
					}

					c.managedResources[name] = append(c.managedResources[name], resType)
				}

				nextResource = resources.NextToken
				if nextResource == nil {
					break
				}
			}
		}

		nextStack = stacks.NextToken
		if nextStack == nil {
			break
		}
	}

	return nil
}

// IsManagedResource checks if the given resource is managed by cloudformation
//
// If PopulateMangedResourceData has been called it will be accurate, however for some accounts this may be slow.
// If PopulateMangedResourceData has not been called it will use a heuristic match, looking for the random ID that
// CFN appends to the name
func (c *cfnClient) IsManagedResource(cfnType CfnResourceType, resourceIdentifier string) bool {
	if c.managedResources != nil {
		return c.managedResources[resourceIdentifier].contains(cfnType)
	}

	if cfnResourceRegexp.MatchString(resourceIdentifier) {
		return true
	}

	return false
}

func (r CfnResourceType) isInterestingResource() bool {
	switch r {
	case CfnIamPolicy, CfnIamRole, CfnIamUser, CfnIamGroup, CfnInstanceProfile, CfnS3Bucket:
		return true
	}

	return false
}

type CfnResourceTypes []CfnResourceType

func (r CfnResourceTypes) contains(t CfnResourceType) bool {
	for _, v := range r {
		if v == t {
			return true
		}
	}
	return false
}
