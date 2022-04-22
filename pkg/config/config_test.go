/*
Copyright 2021 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package config

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"hpc-toolkit/pkg/resreader"

	"github.com/zclconf/go-cty/cty"
	. "gopkg.in/check.v1"
)

var (
	// Shared IO Values
	simpleYamlFilename string
	tmpTestDir         string

	// Expected/Input Values
	expectedYaml = []byte(`
blueprint_name: simple
vars:
  labels:
    ghpc_blueprint: simple
    deployment_name: deployment_name
terraform_backend_defaults:
  type: gcs
  configuration:
    bucket: hpc-toolkit-tf-state
resource_groups:
- group: group1
  resources:
  - source: ./resources/network/vpc
    kind: terraform
    id: "vpc"
    settings:
      network_name: $"${var.deployment_name}_net
      project_id: project_name
`)
	testResources = []Resource{
		{
			Source:           "./resources/network/vpc",
			Kind:             "terraform",
			ID:               "vpc",
			WrapSettingsWith: make(map[string][]string),
			Settings: map[string]interface{}{
				"network_name": "$\"${var.deployment_name}_net\"",
				"project_id":   "project_name",
			},
		},
	}
	defaultLabels = map[string]interface{}{
		"ghpc_blueprint":  "simple",
		"deployment_name": "deployment_name",
	}
	expectedSimpleYamlConfig YamlConfig = YamlConfig{
		BlueprintName:  "simple",
		Vars:           map[string]interface{}{"labels": defaultLabels},
		ResourceGroups: []ResourceGroup{{Name: "ResourceGroup1", TerraformBackend: TerraformBackend{}, Resources: testResources}},
		TerraformBackendDefaults: TerraformBackend{
			Type:          "",
			Configuration: map[string]interface{}{},
		},
	}
	// For expand.go
	requiredVar = resreader.VarInfo{
		Name:        "reqVar",
		Type:        "string",
		Description: "A test required variable",
		Default:     nil,
		Required:    true,
	}
)

// Setup GoCheck
type MySuite struct{}

var _ = Suite(&MySuite{})

func Test(t *testing.T) {
	TestingT(t)
}

// setup opens a temp file to store the yaml and saves it's name
func setup() {
	simpleYamlFile, err := ioutil.TempFile("", "*.yaml")
	if err != nil {
		log.Fatal(err)
	}
	_, err = simpleYamlFile.Write(expectedYaml)
	if err != nil {
		log.Fatal(err)
	}
	simpleYamlFilename = simpleYamlFile.Name()
	simpleYamlFile.Close()

	// Create test directory with simple resources
	tmpTestDir, err = ioutil.TempDir("", "ghpc_config_tests_*")
	if err != nil {
		log.Fatalf("failed to create temp dir for config tests: %e", err)
	}
	resourceDir := filepath.Join(tmpTestDir, "resource")
	err = os.Mkdir(resourceDir, 0755)
	if err != nil {
		log.Fatalf("failed to create test resource dir: %v", err)
	}
	varFile, err := os.Create(filepath.Join(resourceDir, "variables.tf"))
	if err != nil {
		log.Fatalf("failed to create variables.tf in test resource dir: %v", err)
	}
	testVariablesTF := `
	variable "test_variable" {
		description = "Test Variable"
		type        = string
	}`
	_, err = varFile.WriteString(testVariablesTF)
	if err != nil {
		log.Fatalf("failed to write variables.tf in test resource dir: %v", err)
	}
}

// Delete the temp YAML file
func teardown() {
	err := os.Remove(simpleYamlFilename)
	if err != nil {
		log.Fatalf("config_test teardown: %v", err)
	}
	err = os.RemoveAll(tmpTestDir)
	if err != nil {
		log.Fatalf(
			"failed to tear down tmp directory (%s) for config unit tests: %v",
			tmpTestDir, err)
	}
}

// util function
func cleanErrorRegexp(errRegexp string) string {
	errRegexp = strings.ReplaceAll(errRegexp, "[", "\\[")
	errRegexp = strings.ReplaceAll(errRegexp, "]", "\\]")
	return errRegexp
}

func getBlueprintConfigForTest() BlueprintConfig {
	testResourceSource := "testSource"
	testResource := Resource{
		Source:           testResourceSource,
		Kind:             "terraform",
		ID:               "testResource",
		Use:              []string{},
		WrapSettingsWith: make(map[string][]string),
		Settings:         make(map[string]interface{}),
	}
	testResourceSourceWithLabels := "./role/source"
	testResourceWithLabels := Resource{
		Source:           testResourceSourceWithLabels,
		ID:               "testResourceWithLabels",
		Kind:             "terraform",
		Use:              []string{},
		WrapSettingsWith: make(map[string][]string),
		Settings: map[string]interface{}{
			"resourceLabel": "resourceLabelValue",
		},
	}
	testLabelVarInfo := resreader.VarInfo{Name: "labels"}
	testResourceInfo := resreader.ResourceInfo{
		Inputs: []resreader.VarInfo{testLabelVarInfo},
	}
	testYamlConfig := YamlConfig{
		BlueprintName: "simple",
		Validators:    []validatorConfig{},
		Vars:          map[string]interface{}{},
		TerraformBackendDefaults: TerraformBackend{
			Type:          "",
			Configuration: map[string]interface{}{},
		},
		ResourceGroups: []ResourceGroup{
			{
				Name: "group1",
				TerraformBackend: TerraformBackend{
					Type:          "",
					Configuration: map[string]interface{}{},
				},
				Resources: []Resource{testResource, testResourceWithLabels},
			},
		},
	}

	return BlueprintConfig{
		Config: testYamlConfig,
		ResourcesInfo: map[string]map[string]resreader.ResourceInfo{
			"group1": {
				testResourceSource:           testResourceInfo,
				testResourceSourceWithLabels: testResourceInfo,
			},
		},
	}
}

func getBasicBlueprintConfigWithTestResource() BlueprintConfig {
	testResourceSource := filepath.Join(tmpTestDir, "resource")
	testResourceGroup := ResourceGroup{
		Name: "primary",
		Resources: []Resource{
			{
				ID:       "TestResource",
				Kind:     "terraform",
				Source:   testResourceSource,
				Settings: map[string]interface{}{"test_variable": "test_value"},
			},
		},
	}
	return BlueprintConfig{
		Config: YamlConfig{
			Vars:           make(map[string]interface{}),
			ResourceGroups: []ResourceGroup{testResourceGroup},
		},
	}
}

/* Tests */
// config.go
func (s *MySuite) TestExpandConfig(c *C) {
	bc := getBasicBlueprintConfigWithTestResource()
	bc.ExpandConfig()
}

func (s *MySuite) TestSetResourcesInfo(c *C) {
	bc := getBasicBlueprintConfigWithTestResource()
	bc.setResourcesInfo()
}

func (s *MySuite) TestCreateResourceInfo(c *C) {
	bc := getBasicBlueprintConfigWithTestResource()
	createResourceInfo(bc.Config.ResourceGroups[0])
}

func (s *MySuite) TestGetResouceByID(c *C) {
	testID := "testID"

	// No Resources
	rg := ResourceGroup{}
	got := rg.getResourceByID(testID)
	c.Assert(got, DeepEquals, Resource{})

	// No Match
	rg.Resources = []Resource{{ID: "NoMatch"}}
	got = rg.getResourceByID(testID)
	c.Assert(got, DeepEquals, Resource{})

	// Match
	expected := Resource{ID: testID}
	rg.Resources = []Resource{expected}
	got = rg.getResourceByID(testID)
	c.Assert(got, DeepEquals, expected)
}

func (s *MySuite) TestHasKind(c *C) {
	// No resources
	rg := ResourceGroup{}
	c.Assert(rg.HasKind("terraform"), Equals, false)
	c.Assert(rg.HasKind("packer"), Equals, false)
	c.Assert(rg.HasKind("notAKind"), Equals, false)

	// One terraform resources
	rg.Resources = append(rg.Resources, Resource{Kind: "terraform"})
	c.Assert(rg.HasKind("terraform"), Equals, true)
	c.Assert(rg.HasKind("packer"), Equals, false)
	c.Assert(rg.HasKind("notAKind"), Equals, false)

	// Multiple terraform resources
	rg.Resources = append(rg.Resources, Resource{Kind: "terraform"})
	rg.Resources = append(rg.Resources, Resource{Kind: "terraform"})
	c.Assert(rg.HasKind("terraform"), Equals, true)
	c.Assert(rg.HasKind("packer"), Equals, false)
	c.Assert(rg.HasKind("notAKind"), Equals, false)

	// One packer kind
	rg.Resources = []Resource{{Kind: "packer"}}
	c.Assert(rg.HasKind("terraform"), Equals, false)
	c.Assert(rg.HasKind("packer"), Equals, true)
	c.Assert(rg.HasKind("notAKind"), Equals, false)

	// One packer, one terraform
	rg.Resources = append(rg.Resources, Resource{Kind: "terraform"})
	c.Assert(rg.HasKind("terraform"), Equals, true)
	c.Assert(rg.HasKind("packer"), Equals, true)
	c.Assert(rg.HasKind("notAKind"), Equals, false)

}

func (s *MySuite) TestCheckResourceAndGroupNames(c *C) {
	bc := getBlueprintConfigForTest()
	checkResourceAndGroupNames(bc.Config.ResourceGroups)
	testResID := bc.Config.ResourceGroups[0].Resources[0].ID
	c.Assert(bc.ResourceToGroup[testResID], Equals, 0)
}

func (s *MySuite) TestNewBlueprint(c *C) {
	bc := getBlueprintConfigForTest()
	outFile := filepath.Join(tmpTestDir, "out_TestNewBlueprint.yaml")
	bc.ExportYamlConfig(outFile)
	newBC := NewBlueprintConfig(outFile)
	c.Assert(bc.Config, DeepEquals, newBC.Config)
}

func (s *MySuite) TestImportYamlConfig(c *C) {
	obtainedYamlConfig := importYamlConfig(simpleYamlFilename)
	c.Assert(obtainedYamlConfig.BlueprintName,
		Equals, expectedSimpleYamlConfig.BlueprintName)
	c.Assert(
		len(obtainedYamlConfig.Vars["labels"].(map[interface{}]interface{})),
		Equals,
		len(expectedSimpleYamlConfig.Vars["labels"].(map[string]interface{})),
	)
	c.Assert(obtainedYamlConfig.ResourceGroups[0].Resources[0].ID,
		Equals, expectedSimpleYamlConfig.ResourceGroups[0].Resources[0].ID)
}

func (s *MySuite) TestExportYamlConfig(c *C) {
	// Return bytes
	bc := BlueprintConfig{}
	bc.Config = expectedSimpleYamlConfig
	obtainedYaml, err := bc.ExportYamlConfig("")
	c.Assert(err, IsNil)
	c.Assert(obtainedYaml, Not(IsNil))

	// Write file
	outFilename := "out_TestExportYamlConfig.yaml"
	outFile := filepath.Join(tmpTestDir, outFilename)
	bc.ExportYamlConfig(outFile)
	fileInfo, err := os.Stat(outFile)
	c.Assert(err, IsNil)
	c.Assert(fileInfo.Name(), Equals, outFilename)
	c.Assert(fileInfo.Size() > 0, Equals, true)
	c.Assert(fileInfo.IsDir(), Equals, false)
}

func (s *MySuite) TestSetCLIVariables(c *C) {
	// Success
	bc := getBasicBlueprintConfigWithTestResource()
	c.Assert(bc.Config.Vars["project_id"], IsNil)
	c.Assert(bc.Config.Vars["deployment_name"], IsNil)
	c.Assert(bc.Config.Vars["region"], IsNil)
	c.Assert(bc.Config.Vars["zone"], IsNil)

	cliProjectID := "cli_test_project_id"
	cliDeploymentName := "cli_deployment_name"
	cliRegion := "cli_region"
	cliZone := "cli_zone"
	cliKeyVal := "key=val"
	cliVars := []string{
		fmt.Sprintf("project_id=%s", cliProjectID),
		fmt.Sprintf("deployment_name=%s", cliDeploymentName),
		fmt.Sprintf("region=%s", cliRegion),
		fmt.Sprintf("zone=%s", cliZone),
		fmt.Sprintf("kv=%s", cliKeyVal),
	}
	err := bc.SetCLIVariables(cliVars)

	c.Assert(err, IsNil)
	c.Assert(bc.Config.Vars["project_id"], Equals, cliProjectID)
	c.Assert(bc.Config.Vars["deployment_name"], Equals, cliDeploymentName)
	c.Assert(bc.Config.Vars["region"], Equals, cliRegion)
	c.Assert(bc.Config.Vars["zone"], Equals, cliZone)
	c.Assert(bc.Config.Vars["kv"], Equals, cliKeyVal)

	// Failure: Variable without '='
	bc = getBasicBlueprintConfigWithTestResource()
	c.Assert(bc.Config.Vars["project_id"], IsNil)

	invalidNonEQVars := []string{
		fmt.Sprintf("project_id%s", cliProjectID),
	}
	err = bc.SetCLIVariables(invalidNonEQVars)

	expErr := "invalid format: .*"
	c.Assert(err, ErrorMatches, expErr)
	c.Assert(bc.Config.Vars["project_id"], IsNil)
}

func (s *MySuite) TestSetBackendConfig(c *C) {
	// Success
	bc := getBlueprintConfigForTest()
	c.Assert(bc.Config.TerraformBackendDefaults.Type, Equals, "")
	c.Assert(bc.Config.TerraformBackendDefaults.Configuration["bucket"], IsNil)
	c.Assert(bc.Config.TerraformBackendDefaults.Configuration["impersonate_service_account"], IsNil)
	c.Assert(bc.Config.TerraformBackendDefaults.Configuration["prefix"], IsNil)

	cliBEType := "gcs"
	cliBEBucket := "a_bucket"
	cliBESA := "a_bucket_reader@project.iam.gserviceaccount.com"
	cliBEPrefix := "test/prefix"
	cliBEConfigVars := []string{
		fmt.Sprintf("type=%s", cliBEType),
		fmt.Sprintf("bucket=%s", cliBEBucket),
		fmt.Sprintf("impersonate_service_account=%s", cliBESA),
		fmt.Sprintf("prefix=%s", cliBEPrefix),
	}
	err := bc.SetBackendConfig(cliBEConfigVars)

	c.Assert(err, IsNil)
	c.Assert(bc.Config.TerraformBackendDefaults.Type, Equals, cliBEType)
	c.Assert(bc.Config.TerraformBackendDefaults.Configuration["bucket"], Equals, cliBEBucket)
	c.Assert(bc.Config.TerraformBackendDefaults.Configuration["impersonate_service_account"], Equals, cliBESA)
	c.Assert(bc.Config.TerraformBackendDefaults.Configuration["prefix"], Equals, cliBEPrefix)

	// Failure: Variable without '='
	bc = getBlueprintConfigForTest()
	c.Assert(bc.Config.TerraformBackendDefaults.Type, Equals, "")

	invalidNonEQVars := []string{
		fmt.Sprintf("type%s", cliBEType),
		fmt.Sprintf("bucket%s", cliBEBucket),
	}
	err = bc.SetBackendConfig(invalidNonEQVars)

	expErr := "invalid format: .*"
	c.Assert(err, ErrorMatches, expErr)
}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	teardown()
	os.Exit(code)
}

func (s *MySuite) TestValidationLevels(c *C) {
	var err error
	var ok bool
	bc := getBlueprintConfigForTest()
	validLevels := []string{"ERROR", "WARNING", "IGNORE"}
	for idx, level := range validLevels {
		err = bc.SetValidationLevel(level)
		c.Assert(err, IsNil)
		ok = isValidValidationLevel(idx)
		c.Assert(ok, Equals, true)
	}

	err = bc.SetValidationLevel("INVALID")
	c.Assert(err, NotNil)

	// check that our test for iota enum is working
	ok = isValidValidationLevel(-1)
	c.Assert(ok, Equals, false)
	invalidLevel := len(validLevels) + 1
	ok = isValidValidationLevel(invalidLevel)
	c.Assert(ok, Equals, false)
}

func (s *MySuite) TestIsLiteralVariable(c *C) {
	var matched bool
	matched = IsLiteralVariable("((var.project_id))")
	c.Assert(matched, Equals, true)
	matched = IsLiteralVariable("(( var.project_id ))")
	c.Assert(matched, Equals, true)
	matched = IsLiteralVariable("(var.project_id)")
	c.Assert(matched, Equals, false)
	matched = IsLiteralVariable("var.project_id")
	c.Assert(matched, Equals, false)
}

func (s *MySuite) TestIdentifyLiteralVariable(c *C) {
	var ctx, name string
	var ok bool
	ctx, name, ok = IdentifyLiteralVariable("((var.project_id))")
	c.Assert(ctx, Equals, "var")
	c.Assert(name, Equals, "project_id")
	c.Assert(ok, Equals, true)

	ctx, name, ok = IdentifyLiteralVariable("((module.structure.nested_value))")
	c.Assert(ctx, Equals, "module")
	c.Assert(name, Equals, "structure.nested_value")
	c.Assert(ok, Equals, true)

	// TODO: properly variables with periods in them!
	// One purpose of literal variables is to refer to values in nested
	// structures of a module output; should probably accept that case
	// but not global variables with periods in them
	ctx, name, ok = IdentifyLiteralVariable("var.project_id")
	c.Assert(ctx, Equals, "")
	c.Assert(name, Equals, "")
	c.Assert(ok, Equals, false)
}

func (s *MySuite) TestConvertToCty(c *C) {
	var testval interface{}
	var testcty cty.Value
	var err error

	testval = "test"
	testcty, err = ConvertToCty(testval)
	c.Assert(testcty.Type(), Equals, cty.String)
	c.Assert(err, IsNil)

	testval = complex(1, -1)
	testcty, err = ConvertToCty(testval)
	c.Assert(testcty.Type(), Equals, cty.NilType)
	c.Assert(err, NotNil)
}

func (s *MySuite) TestConvertMapToCty(c *C) {
	var testmap map[string]interface{}
	var testcty map[string]cty.Value
	var err error
	var testkey = "testkey"
	var testval = "testval"
	testmap = map[string]interface{}{
		testkey: testval,
	}

	testcty, err = ConvertMapToCty(testmap)
	c.Assert(err, IsNil)
	ctyval, found := testcty[testkey]
	c.Assert(found, Equals, true)
	c.Assert(ctyval.Type(), Equals, cty.String)

	testmap = map[string]interface{}{
		"testkey": complex(1, -1),
	}
	testcty, err = ConvertMapToCty(testmap)
	c.Assert(err, NotNil)
	ctyval, found = testcty[testkey]
	c.Assert(found, Equals, false)
}

func (s *MySuite) TestResolveGlobalVariables(c *C) {
	var err error
	var testkey1 = "testkey1"
	var testkey2 = "testkey2"
	var testkey3 = "testkey3"
	bc := getBlueprintConfigForTest()
	ctyMap := make(map[string]cty.Value)
	err = bc.Config.ResolveGlobalVariables(ctyMap)
	c.Assert(err, IsNil)

	// confirm plain string is unchanged and does not error
	testCtyString := cty.StringVal("testval")
	ctyMap[testkey1] = testCtyString
	err = bc.Config.ResolveGlobalVariables(ctyMap)
	c.Assert(err, IsNil)
	c.Assert(ctyMap[testkey1], Equals, testCtyString)

	// confirm literal, non-global, variable is unchanged and does not error
	testCtyString = cty.StringVal("((module.testval))")
	ctyMap[testkey1] = testCtyString
	err = bc.Config.ResolveGlobalVariables(ctyMap)
	c.Assert(err, IsNil)
	c.Assert(ctyMap[testkey1], Equals, testCtyString)

	// confirm failed resolution of a literal global
	testCtyString = cty.StringVal("((var.test_global_var))")
	ctyMap[testkey1] = testCtyString
	err = bc.Config.ResolveGlobalVariables(ctyMap)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, ".*Unsupported attribute;.*")

	// confirm successful resolution of literal globals in presence of other strings
	testGlobalVarString := "test_global_string"
	testGlobalValString := "testval"
	testGlobalVarBool := "test_global_bool"
	testGlobalValBool := "testval"
	testPlainString := "plain-string"
	bc.Config.Vars[testGlobalVarString] = testGlobalValString
	bc.Config.Vars[testGlobalVarBool] = testGlobalValBool
	testCtyString = cty.StringVal(fmt.Sprintf("((var.%s))", testGlobalVarString))
	testCtyBool := cty.StringVal(fmt.Sprintf("((var.%s))", testGlobalVarBool))
	ctyMap[testkey1] = testCtyString
	ctyMap[testkey2] = testCtyBool
	ctyMap[testkey3] = cty.StringVal(testPlainString)
	err = bc.Config.ResolveGlobalVariables(ctyMap)
	c.Assert(err, IsNil)
	c.Assert(ctyMap[testkey1], Equals, cty.StringVal(testGlobalValString))
	c.Assert(ctyMap[testkey2], Equals, cty.StringVal(testGlobalValBool))
	c.Assert(ctyMap[testkey3], Equals, cty.StringVal(testPlainString))
}
