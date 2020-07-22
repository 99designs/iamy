package iamy

import "testing"

func TestCfnMangedResources(t *testing.T) {
	t.Run("With fetched CFN resource lists", func(t *testing.T) {

		cfn := cfnClient{
			managedResources: map[string]CfnResourceTypes{
				"foobar": []CfnResourceType{CfnIamPolicy, CfnIamRole},
			},
		}

		if cfn.IsManagedResource(CfnIamUser, "foobar") {
			t.Fatal("different object types with same name is not managed")
		}

		if !cfn.IsManagedResource(CfnIamPolicy, "foobar") {
			t.Fatal("matching object and type should be managed")
		}
	})

	t.Run("With heuristic matching", func(t *testing.T) {
		cfn := cfnClient{}

		if cfn.IsManagedResource(CfnIamUser, "foobar") {
			t.Fatal("names without id suffix are not managed")
		}

		if !cfn.IsManagedResource(CfnIamPolicy, "foobar-ABCDEFGH1234567") {
			t.Fatal("names with id suffix are managed")
		}
	})
}
