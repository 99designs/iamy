package iamy

import (
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/pkg/errors"
)

const NoSuchBucketPolicyErrCode = "NoSuchBucketPolicy"

func newRegionClientMap(s *session.Session) *regionClientMap {
	return &regionClientMap{
		clients: map[string]s3iface.S3API{},
		sess:    s,
		mutex:   &sync.Mutex{},
	}
}

type regionClientMap struct {
	sess    *session.Session
	clients map[string]s3iface.S3API
	mutex   *sync.Mutex
}

func (scm *regionClientMap) add(c *s3.S3) {
	scm.clients[*c.Config.Region] = c
}

func (scm *regionClientMap) getOrCreate(region string) s3iface.S3API {
	scm.mutex.Lock()
	if _, ok := scm.clients[region]; !ok {
		scm.clients[region] = s3.New(scm.sess, aws.NewConfig().WithRegion(region))
	}
	scm.mutex.Unlock()

	return scm.clients[region]
}

type s3Client struct {
	s3iface.S3API
	regionClients *regionClientMap
}

func newS3Client(s *session.Session) *s3Client {
	defaultClient := s3.New(s)
	clients := newRegionClientMap(s)
	clients.add(defaultClient)

	return &s3Client{
		S3API:         defaultClient,
		regionClients: clients,
	}
}

type bucket struct {
	name       string
	policyJson string
	exists     bool
}

func (c *s3Client) withRegion(region string) s3iface.S3API {
	if region == "" {
		return c.S3API
	}

	return c.regionClients.getOrCreate(region)
}

func normaliseString(a *string) (b string) {
	if a != nil {
		b = *a
	}
	return
}

func (c *s3Client) populateBucket(b *bucket) error {
	r, err := c.GetBucketLocation(&s3.GetBucketLocationInput{Bucket: aws.String(b.name)})
	if err != nil {
		return err
	}

	region := s3.NormalizeBucketLocation(normaliseString(r.LocationConstraint))
	b.policyJson, err = c.GetBucketPolicyDoc(b.name, region)

	return err
}

func (c *s3Client) listAllBuckets() ([]*bucket, error) {
	bucketListResp, err := c.ListBuckets(&s3.ListBucketsInput{})
	if err != nil {
		return nil, errors.Wrap(err, "Error while calling ListBuckets")
	}

	var wg sync.WaitGroup
	var oneOfTheErrorsDuringPopulation error
	buckets := []*bucket{}

	for _, rb := range bucketListResp.Buckets {
		b := bucket{name: *rb.Name}
		buckets = append(buckets, &b)

		wg.Add(1)
		go func() {
			defer wg.Done()
			err := c.populateBucket(&b)
			if err != nil {
				if awsErr, ok := err.(awserr.Error); ok {
					if awsErr.Code() != s3.ErrCodeNoSuchBucket {
						oneOfTheErrorsDuringPopulation = errors.New(fmt.Sprintf("Error while getting details for S3 bucket %s: %s", b.name, err))
					}
				}
			} else {
				b.exists = true
			}
		}()
	}
	wg.Wait()

	bucketsExist := []*bucket{}

	for _, b := range buckets {
		if b.exists {
			bucketsExist = append(bucketsExist, b)
		}
	}

	if oneOfTheErrorsDuringPopulation != nil {
		return nil, oneOfTheErrorsDuringPopulation
	}

	return bucketsExist, nil
}

func (c *s3Client) GetBucketPolicyDoc(name, region string) (string, error) {
	clientForRegion := c.withRegion(region)
	resp, err := clientForRegion.GetBucketPolicy(&s3.GetBucketPolicyInput{
		Bucket: aws.String(name),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == NoSuchBucketPolicyErrCode {
				return "", nil
			}
		}
		return "", fmt.Errorf("GetBucketPolicyDoc for %s: %s", name, err.Error())
	}

	return *resp.Policy, nil
}
