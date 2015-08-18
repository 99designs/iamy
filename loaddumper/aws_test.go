package loaddumper

import (
	"fmt"
	"testing"

	"github.com/99designs/iamy/loaddumper/mock_iamiface"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/golang/mock/gomock"
)

// sample data
var (
	testUser = iam.User{
		Arn:      aws.String("arn:aws:iam::012345678901:user/testuser"),
		Path:     aws.String("/testpath"),
		UserName: aws.String("testuser"),
	}
	testGroup = iam.Group{
		Arn:       aws.String("arn:aws:iam::012345678901:group/testgroup"),
		GroupName: aws.String("testgroup"),
		Path:      aws.String("/testpath"),
	}
	testPolicy = iam.Policy{
		Arn:          aws.String("arn:aws:iam::012345678901:policy/testpolicy"),
		PolicyName:   aws.String("testpolicy"),
		Path:         aws.String("/testpath"),
		IsAttachable: aws.Bool(true),
	}
	testAttachedPolicy = iam.AttachedPolicy{
		PolicyArn:  aws.String("arn:aws:iam::012345678901:policy/testattachedpolicy"),
		PolicyName: aws.String("testattachedpolicy"),
	}
	testRole = iam.Role{
		Arn:      aws.String("arn:aws:iam::012345678901:role/testrole"),
		RoleName: aws.String("testrole"),
		Path:     aws.String("/testpath"),
		AssumeRolePolicyDocument: testPolicyDocument,
	}
	testPolicyVersion = iam.PolicyVersion{
		IsDefaultVersion: aws.Bool(true),
		VersionId:        aws.String("2"),
		Document:         aws.String(`{ "testkey": "testPolicyVersionDocument" }`),
	}
	testPolicyDocument = aws.String(`{ "testkey": "TestPolicyDocument" }`)
)

func TestLoad(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockClient := mock_iamiface.NewMockIAMAPI(mockCtrl)
	mockClient.EXPECT().
		GetUser(gomock.Any()).
		Return(&iam.GetUserOutput{User: &testUser}, nil)
	mockClient.EXPECT().
		ListAccountAliases(gomock.Any()).
		Return(&iam.ListAccountAliasesOutput{AccountAliases: []*string{aws.String("testalias")}}, nil)
	mockClient.EXPECT().
		ListUsers(gomock.Any()).
		Return(&iam.ListUsersOutput{Users: []*iam.User{&testUser}}, nil)
	mockClient.EXPECT().
		ListGroupsForUser(gomock.Any()).
		Return(&iam.ListGroupsForUserOutput{Groups: []*iam.Group{&testGroup}}, nil)
	mockClient.EXPECT().
		ListUserPolicies(gomock.Any()).
		Return(&iam.ListUserPoliciesOutput{PolicyNames: []*string{testPolicy.PolicyName}}, nil)
	mockClient.EXPECT().
		GetUserPolicy(gomock.Any()).
		Return(&iam.GetUserPolicyOutput{PolicyDocument: testPolicyDocument, PolicyName: testPolicy.PolicyName}, nil)
	mockClient.EXPECT().
		ListAttachedUserPolicies(gomock.Any()).
		Return(&iam.ListAttachedUserPoliciesOutput{AttachedPolicies: []*iam.AttachedPolicy{&testAttachedPolicy}}, nil)
	mockClient.EXPECT().
		ListPolicies(gomock.Any()).
		Return(&iam.ListPoliciesOutput{Policies: []*iam.Policy{&testPolicy}}, nil)
	mockClient.EXPECT().
		ListPolicyVersions(gomock.Any()).
		Return(&iam.ListPolicyVersionsOutput{Versions: []*iam.PolicyVersion{&testPolicyVersion}}, nil)
	mockClient.EXPECT().
		GetPolicyVersion(gomock.Any()).
		Return(&iam.GetPolicyVersionOutput{PolicyVersion: &testPolicyVersion}, nil)
	mockClient.EXPECT().
		ListGroups(gomock.Any()).
		Return(&iam.ListGroupsOutput{Groups: []*iam.Group{&testGroup}}, nil)
	mockClient.EXPECT().
		ListGroupPolicies(gomock.Any()).
		Return(&iam.ListGroupPoliciesOutput{PolicyNames: []*string{testPolicy.PolicyName}}, nil)
	mockClient.EXPECT().
		GetGroupPolicy(gomock.Any()).
		Return(&iam.GetGroupPolicyOutput{PolicyDocument: testPolicyDocument, PolicyName: testPolicy.PolicyName}, nil)
	mockClient.EXPECT().
		ListAttachedGroupPolicies(gomock.Any()).
		Return(&iam.ListAttachedGroupPoliciesOutput{AttachedPolicies: []*iam.AttachedPolicy{&testAttachedPolicy}}, nil)
	mockClient.EXPECT().
		ListRoles(gomock.Any()).
		Return(&iam.ListRolesOutput{Roles: []*iam.Role{&testRole}}, nil)
	mockClient.EXPECT().
		ListRolePolicies(gomock.Any()).
		Return(&iam.ListRolePoliciesOutput{PolicyNames: []*string{testPolicy.PolicyName}}, nil)
	mockClient.EXPECT().
		GetRolePolicy(gomock.Any()).
		Return(&iam.GetRolePolicyOutput{PolicyDocument: testPolicyDocument, PolicyName: testPolicy.PolicyName}, nil)
	mockClient.EXPECT().
		ListAttachedRolePolicies(gomock.Any()).
		Return(&iam.ListAttachedRolePoliciesOutput{AttachedPolicies: []*iam.AttachedPolicy{&testAttachedPolicy}}, nil)

	Aws.client = mockClient

	// test that the models that Aws.Load creates
	data, err := Aws.Load()
	if err != nil {
		t.Fatal(err.Error())
	}

	expected := `[{Account:testalias-012345678901 Users:[{Name:testuser Path:/testpath Groups:[testgroup] InlinePolicies:[{Name:testpolicy Policy:map[testkey:TestPolicyDocument]}] Policies:[testattachedpolicy]}] Groups:[{Name:testgroup Path: Roles:[] InlinePolicies:[{Name:testpolicy Policy:map[testkey:TestPolicyDocument]}] Policies:[testattachedpolicy]}] Roles:[{Name:testrole Path: AssumeRolePolicyDocument:map[testkey:TestPolicyDocument] InlinePolicies:[{Name:testpolicy Policy:map[testkey:TestPolicyDocument]}] Policies:[testattachedpolicy]}] Policies:[{Name:testpolicy Path:/testpath IsAttachable:true Version:2 Policy:map[testkey:testPolicyVersionDocument]}]}]`
	actual := fmt.Sprintf("%+v", data)

	if actual != expected {
		t.Errorf("Expected \n--\n%s\n--\n, got \n--\n%s\n--\n", expected, actual)
	}
}
