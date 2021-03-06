// Copyright 2015-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//	http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package image

import (
	"errors"
	"flag"
	"os"
	"testing"
	"time"

	"github.com/aws/amazon-ecs-cli/ecs-cli/modules/clients/aws/ecr"
	"github.com/aws/amazon-ecs-cli/ecs-cli/modules/clients/aws/ecr/mock"
	"github.com/aws/amazon-ecs-cli/ecs-cli/modules/clients/aws/sts/mock"
	"github.com/aws/amazon-ecs-cli/ecs-cli/modules/clients/aws/tagging/mock"
	"github.com/aws/amazon-ecs-cli/ecs-cli/modules/clients/docker/mock"
	"github.com/aws/amazon-ecs-cli/ecs-cli/modules/commands/flags"
	"github.com/aws/amazon-ecs-cli/ecs-cli/modules/config"
	"github.com/aws/aws-sdk-go/aws"
	ecrApi "github.com/aws/aws-sdk-go/service/ecr"
	taggingSDK "github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/fsouza/go-dockerclient"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
)

const (
	repository          = "repository"
	repositoryWithSlash = "hi/repo"
	tag                 = "tag-v0.1.0"
	image               = repository + ":" + tag
	registry            = "https://" + registryID + ".dkr.ecr.us-west-2.amazonaws.com"
	region              = "us-west-2"
	registryID          = "012345678912"
	repositoryURI       = registry + "/" + repository
	clusterName         = "defaultCluster"
)

type mockReadWriter struct {
	clusterName string
}

func (rdwr *mockReadWriter) Get(cluster string, profile string) (*config.LocalConfig, error) {
	return config.NewLocalConfig(rdwr.clusterName), nil
}

func (rdwr *mockReadWriter) SaveProfile(configName string, profile *config.Profile) error {
	return nil
}

func (rdwr *mockReadWriter) SaveCluster(configName string, cluster *config.Cluster) error {
	return nil
}

func (rdwr *mockReadWriter) SetDefaultProfile(configName string) error {
	return nil
}

func (rdwr *mockReadWriter) SetDefaultCluster(configName string) error {
	return nil
}

func newMockReadWriter() *mockReadWriter {
	return &mockReadWriter{clusterName: clusterName}
}

func TestImagePush(t *testing.T) {
	mockECR, mockDocker, mockSTS, mockTagging := setupTestController(t)
	setupEnvironmentVar()

	gomock.InOrder(
		mockSTS.EXPECT().GetAWSAccountID().Return(registryID, nil),
		mockECR.EXPECT().GetAuthorizationTokenByID(gomock.Any()).Return(&ecr.Auth{
			Registry: registry,
		}, nil),
		mockDocker.EXPECT().TagImage(image, repositoryURI, tag).Return(nil),
		mockECR.EXPECT().RepositoryExists(repository).Return(false),
		mockECR.EXPECT().CreateRepository(repository).Return(repository, nil),
		mockDocker.EXPECT().PushImage(repositoryURI, tag, registry,
			docker.AuthConfiguration{}).Return(nil),
	)

	context := setAllPushImageFlags()
	err := pushImage(context, region, mockDocker, mockECR, mockSTS, mockTagging)
	assert.NoError(t, err, "Error pushing image")
}

func TestImagePush_WithTags(t *testing.T) {
	mockECR, mockDocker, mockSTS, mockTagging := setupTestController(t)
	setupEnvironmentVar()

	expectedTags := map[string]*string{
		"Hey":         aws.String("You"),
		"Comfortably": aws.String("Numb"),
		"The":         aws.String("Wall"),
	}

	gomock.InOrder(
		mockSTS.EXPECT().GetAWSAccountID().Return(registryID, nil),
		mockECR.EXPECT().GetAuthorizationTokenByID(gomock.Any()).Return(&ecr.Auth{
			Registry: registry,
		}, nil),
		mockDocker.EXPECT().TagImage(image, repositoryURI, tag).Return(nil),
		mockECR.EXPECT().RepositoryExists(repository).Return(false),
		mockECR.EXPECT().CreateRepository(repository).Return(repository, nil),
		mockTagging.EXPECT().TagResources(gomock.Any()).Do(func(x interface{}) {
			input := x.(*taggingSDK.TagResourcesInput)
			assert.Equal(t, expectedTags, input.Tags, "Expected tags to match")
		}).Return(&taggingSDK.TagResourcesOutput{}, nil),
		mockDocker.EXPECT().PushImage(repositoryURI, tag, registry,
			docker.AuthConfiguration{}).Return(nil),
	)

	flagSet := flag.NewFlagSet("ecs-cli-push", 0)
	flagSet.Parse([]string{image})
	flagSet.String(flags.ResourceTagsFlag, "Hey=You,Comfortably=Numb,The=Wall", "")
	context := cli.NewContext(nil, flagSet, nil)
	err := pushImage(context, region, mockDocker, mockECR, mockSTS, mockTagging)
	assert.NoError(t, err, "Error pushing image")
}

func TestImagePushWithURI(t *testing.T) {
	repositoryWithURI := "012345678912.dkr.ecr.us-east-1.amazonaws.com/" + image

	mockECR, mockDocker, mockSTS, mockTagging := setupTestController(t)
	setupEnvironmentVar()

	gomock.InOrder(
		// Skips GetAWSAccountID
		mockECR.EXPECT().GetAuthorizationToken(gomock.Any()).Return(&ecr.Auth{
			Registry: registry,
		}, nil),
		// Skips TagImage
		mockECR.EXPECT().RepositoryExists(repository).Return(false),
		mockECR.EXPECT().CreateRepository(repository).Return(repository, nil),
		mockDocker.EXPECT().PushImage(repositoryURI, tag, registry,
			docker.AuthConfiguration{}).Return(nil),
	)

	flagSet := flag.NewFlagSet("ecs-cli-push", 0)
	flagSet.Parse([]string{repositoryWithURI})
	context := cli.NewContext(nil, flagSet, nil)
	err := pushImage(context, region, mockDocker, mockECR, mockSTS, mockTagging)
	assert.NoError(t, err, "Error pushing image")
}

func TestImagePushWhenRepositoryExists(t *testing.T) {
	mockECR, mockDocker, mockSTS, mockTagging := setupTestController(t)
	setupEnvironmentVar()

	gomock.InOrder(
		mockSTS.EXPECT().GetAWSAccountID().Return(registryID, nil),
		mockECR.EXPECT().GetAuthorizationTokenByID(gomock.Any()).Return(&ecr.Auth{
			Registry: registry,
		}, nil),
		mockDocker.EXPECT().TagImage(image, repositoryURI, tag).Return(nil),
		mockECR.EXPECT().RepositoryExists(repository).Return(true),
		// Skips CreateRepository
		mockDocker.EXPECT().PushImage(repositoryURI, tag, registry,
			docker.AuthConfiguration{}).Return(nil),
	)

	context := setAllPushImageFlags()
	err := pushImage(context, region, mockDocker, mockECR, mockSTS, mockTagging)
	assert.NoError(t, err, "Error pushing image")
}

func TestImagePushWithNoArguments(t *testing.T) {
	mockECR, mockDocker, mockSTS, mockTagging := setupTestController(t)
	setupEnvironmentVar()

	flagSet := flag.NewFlagSet("ecs-cli-push", 0)
	context := cli.NewContext(nil, flagSet, nil)
	err := pushImage(context, region, mockDocker, mockECR, mockSTS, mockTagging)
	assert.Error(t, err, "Expect error pushing image")
}

func TestImagePushWithTooManyArguments(t *testing.T) {
	mockECR, mockDocker, mockSTS, mockTagging := setupTestController(t)
	setupEnvironmentVar()

	flagSet := flag.NewFlagSet("ecs-cli-push", 0)
	flagSet.Parse([]string{repository, image})
	context := cli.NewContext(nil, flagSet, nil)
	err := pushImage(context, region, mockDocker, mockECR, mockSTS, mockTagging)
	assert.Error(t, err, "Expect error pushing image")
}

func TestImagePushWhenGethAuthorizationTokenFail(t *testing.T) {
	mockECR, mockDocker, mockSTS, mockTagging := setupTestController(t)
	setupEnvironmentVar()

	gomock.InOrder(
		mockSTS.EXPECT().GetAWSAccountID().Return(registryID, nil),
		mockECR.EXPECT().GetAuthorizationTokenByID(gomock.Any()).Return(nil, errors.New("something failed")),
	)

	context := setAllPushImageFlags()
	err := pushImage(context, region, mockDocker, mockECR, mockSTS, mockTagging)
	assert.Error(t, err, "Expect error pushing image")
}

func TestImagePushWhenTagImageFail(t *testing.T) {
	mockECR, mockDocker, mockSTS, mockTagging := setupTestController(t)
	setupEnvironmentVar()

	gomock.InOrder(
		mockSTS.EXPECT().GetAWSAccountID().Return(registryID, nil),
		mockECR.EXPECT().GetAuthorizationTokenByID(gomock.Any()).Return(&ecr.Auth{
			Registry: registry,
		}, nil),
		mockDocker.EXPECT().TagImage(image, repositoryURI, tag).Return(errors.New("something failed")),
	)

	context := setAllPushImageFlags()
	err := pushImage(context, region, mockDocker, mockECR, mockSTS, mockTagging)
	assert.Error(t, err, "Expect error pushing image")
}

func TestImagePushWhenCreateRepositoryFail(t *testing.T) {
	mockECR, mockDocker, mockSTS, mockTagging := setupTestController(t)
	setupEnvironmentVar()

	gomock.InOrder(
		mockSTS.EXPECT().GetAWSAccountID().Return(registryID, nil),
		mockECR.EXPECT().GetAuthorizationTokenByID(gomock.Any()).Return(&ecr.Auth{
			Registry: registry,
		}, nil),
		mockDocker.EXPECT().TagImage(image, repositoryURI, tag).Return(nil),
		mockECR.EXPECT().RepositoryExists(repository).Return(false),
		mockECR.EXPECT().CreateRepository(repository).Return("", errors.New("something failed")),
	)

	context := setAllPushImageFlags()
	err := pushImage(context, region, mockDocker, mockECR, mockSTS, mockTagging)
	assert.Error(t, err, "Expect error pushing image")
}

func TestImagePushFail(t *testing.T) {
	mockECR, mockDocker, mockSTS, mockTagging := setupTestController(t)
	setupEnvironmentVar()

	gomock.InOrder(
		mockSTS.EXPECT().GetAWSAccountID().Return(registryID, nil),
		mockECR.EXPECT().GetAuthorizationTokenByID(gomock.Any()).Return(&ecr.Auth{
			Registry: registry,
		}, nil),
		mockDocker.EXPECT().TagImage(image, repositoryURI, tag).Return(nil),
		mockECR.EXPECT().RepositoryExists(repository).Return(false),
		mockECR.EXPECT().CreateRepository(repository).Return(repository, nil),
		mockDocker.EXPECT().PushImage(repositoryURI, tag, registry,
			docker.AuthConfiguration{}).Return(errors.New("something failed")),
	)

	context := setAllPushImageFlags()
	err := pushImage(context, region, mockDocker, mockECR, mockSTS, mockTagging)
	assert.Error(t, err, "Expect error pushing image")
}

func TestImagePull(t *testing.T) {
	mockECR, mockDocker, mockSTS, _ := setupTestController(t)
	setupEnvironmentVar()

	gomock.InOrder(
		mockSTS.EXPECT().GetAWSAccountID().Return(registryID, nil),
		mockECR.EXPECT().GetAuthorizationTokenByID(gomock.Any()).Return(&ecr.Auth{
			Registry: registry,
		}, nil),
		mockDocker.EXPECT().PullImage(repositoryURI, tag,
			docker.AuthConfiguration{}).Return(nil),
	)

	context := setAllPullImageFlags()
	err := pullImage(context, newMockReadWriter(), mockDocker, mockECR, mockSTS)
	assert.NoError(t, err, "Error pulling image")
}

func TestImagePullWithoutImage(t *testing.T) {
	mockECR, mockDocker, mockSTS, _ := setupTestController(t)
	setupEnvironmentVar()

	flagSet := flag.NewFlagSet("ecs-cli-pull", 0)
	context := cli.NewContext(nil, flagSet, nil)
	err := pullImage(context, newMockReadWriter(), mockDocker, mockECR, mockSTS)
	assert.Error(t, err, "Expected error pulling image")
}

func TestImagePullWhenGetAuthorizationTokenFail(t *testing.T) {
	mockECR, mockDocker, mockSTS, _ := setupTestController(t)
	setupEnvironmentVar()

	gomock.InOrder(
		mockSTS.EXPECT().GetAWSAccountID().Return(registryID, nil),
		mockECR.EXPECT().GetAuthorizationTokenByID(gomock.Any()).Return(nil, errors.New("something failed")),
	)

	context := setAllPullImageFlags()
	err := pullImage(context, newMockReadWriter(), mockDocker, mockECR, mockSTS)
	assert.Error(t, err, "Expected error pulling image")
}

func TestImagePullFail(t *testing.T) {
	mockECR, mockDocker, mockSTS, _ := setupTestController(t)
	setupEnvironmentVar()

	gomock.InOrder(
		mockSTS.EXPECT().GetAWSAccountID().Return(registryID, nil),
		mockECR.EXPECT().GetAuthorizationTokenByID(gomock.Any()).Return(&ecr.Auth{
			Registry: registry,
		}, nil),
		mockDocker.EXPECT().PullImage(repositoryURI, tag,
			docker.AuthConfiguration{}).Return(errors.New("something failed")),
	)

	context := setAllPullImageFlags()
	err := pullImage(context, newMockReadWriter(), mockDocker, mockECR, mockSTS)
	assert.Error(t, err, "Expected error pulling image")
}

func TestImageList(t *testing.T) {
	mockECR, _, _, _ := setupTestController(t)
	setupEnvironmentVar()

	imageDigest := "sha:2561234567"
	repositoryName := "repo-name"
	pushedAt := time.Unix(1489687380, 0)
	size := int64(1024)
	tags := aws.StringSlice([]string{"tag1", "tag2"})
	gomock.InOrder(
		mockECR.EXPECT().GetImages(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Do(func(_, _, _, x interface{}) {
			funct := x.(ecr.ProcessImageDetails)
			funct([]*ecrApi.ImageDetail{&ecrApi.ImageDetail{
				ImageDigest:      aws.String(imageDigest),
				RepositoryName:   aws.String(repositoryName),
				ImagePushedAt:    &pushedAt,
				ImageSizeInBytes: aws.Int64(size),
				ImageTags:        tags,
			}})
		}).Return(nil),
	)

	flagSet := flag.NewFlagSet("ecs-cli-images", 0)
	context := cli.NewContext(nil, flagSet, nil)
	err := getImages(context, newMockReadWriter(), mockECR)
	assert.NoError(t, err, "Error listing images")
}

func TestImageListFail(t *testing.T) {
	mockECR, _, _, _ := setupTestController(t)
	setupEnvironmentVar()

	gomock.InOrder(
		mockECR.EXPECT().GetImages(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("something failed")),
	)

	flagSet := flag.NewFlagSet("ecs-cli-images", 0)
	context := cli.NewContext(nil, flagSet, nil)
	err := getImages(context, newMockReadWriter(), mockECR)
	assert.Error(t, err, "Expected error listing images")
}

func TestSplitImageName(t *testing.T) {
	tests := []struct {
		name       string
		uri        string
		repository string
		tag        string
		sha        string
	}{
		{
			name:       "With tag",
			uri:        "",
			repository: repository,
			tag:        tag,
			sha:        "",
		},
		{
			name:       "With SHA 256",
			uri:        "",
			repository: repository,
			tag:        "",
			sha:        "sha256:0b3787ac21ffb4edbd6710e0e60f991d5ded8d8a4f558209ef5987f73db4211a",
		},
		{
			name:       "With URI",
			uri:        "012345678912.dkr.ecr.us-east-1.amazonaws.com",
			repository: repository,
			tag:        "",
			sha:        "",
		},
		{
			name:       "Repository With a Slash in Image Name",
			uri:        "",
			repository: repositoryWithSlash,
			tag:        "",
			sha:        "",
		},
		{
			name:       "Repository with Slash In Image Name and Tag",
			uri:        "",
			repository: repositoryWithSlash,
			tag:        tag,
			sha:        "",
		},
		{
			name:       "Repository With a Slash in Image Name and SHA",
			uri:        "",
			repository: repositoryWithSlash,
			tag:        "",
			sha:        "sha256:0b3787ac21ffb4edbd6710e0e60f991d5ded8d8a4f558209ef5987f73db4211a",
		},
		{
			name:       "Repository with Slash In Image Name and URI",
			uri:        "012345678912.dkr.ecr.us-east-1.amazonaws.com",
			repository: repositoryWithSlash,
			tag:        "",
			sha:        "",
		},
		{
			name:       "Repository with Slash In Image Name and URI and Tag",
			uri:        "012345678912.dkr.ecr.us-east-1.amazonaws.com",
			repository: repositoryWithSlash,
			tag:        tag,
			sha:        "",
		},
		{
			name:       "Repository with Slash In Image Name and URI and Sha256",
			uri:        "012345678912.dkr.ecr.us-east-1.amazonaws.com",
			repository: repositoryWithSlash,
			tag:        "",
			sha:        "sha256:0b3787ac21ffb4edbd6710e0e60f991d5ded8d8a4f558209ef5987f73db4211a",
		},
		{
			name:       "Using FIPS endpoint",
			uri:        "012345678912.dkr.ecr-fips.us-gov-west-1.amazonaws.com",
			repository: repository,
			tag:        "",
			sha:        "",
		},
		{
			name:       "Using FIPS endpoint and slash in image name",
			uri:        "012345678912.dkr.ecr-fips.us-gov-west-1.amazonaws.com",
			repository: repositoryWithSlash,
			tag:        "",
			sha:        "",
		},
		{
			name:       "Using FIPS endpoint and slash in image name and tag",
			uri:        "012345678912.dkr.ecr-fips.us-gov-west-1.amazonaws.com",
			repository: repositoryWithSlash,
			tag:        tag,
			sha:        "",
		},
		{
			name:       "Using FIPS endpoint and slash in image name and sha",
			uri:        "012345678912.dkr.ecr-fips.us-gov-west-1.amazonaws.com",
			repository: repositoryWithSlash,
			tag:        "",
			sha:        "sha256:0b3787ac21ffb4edbd6710e0e60f991d5ded8d8a4f558209ef5987f73db4211a",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			expectedImage := test.repository

			if test.uri != "" {
				expectedImage = test.uri + "/" + expectedImage
			}
			if test.tag != "" {
				expectedImage += ":" + test.tag
			}
			if test.sha != "" {
				expectedImage += "@" + test.sha
			}

			observedRegistryURI, observedRepo, observedTag, err := splitImageName(expectedImage, "[:|@]", "format")
			assert.Equal(t, test.uri, observedRegistryURI, "RegistryURI should match")
			assert.Equal(t, test.repository, observedRepo, "Repository should match")

			// Can only specify either tag or sha
			if test.tag != "" {
				assert.Equal(t, test.tag, observedTag, "Tag should match")
			}
			if test.sha != "" {
				assert.Equal(t, test.sha, observedTag, "SHA should match")
			}
			assert.NoError(t, err, "Error splitting image name")
		})
	}
}

func TestSplitImageNameErrorCaseBadURI(t *testing.T) {
	badURI := "012345678912.dkr.ecr-blips.us-gov-west-1.amazonaws.com"
	invalidImage := badURI + "/" + repository
	_, _, _, err := splitImageName(invalidImage, "[:]", "format")

	assert.Error(t, err, "Expected error splitting image name")
}

func TestSplitImageNameErrorCase(t *testing.T) {
	invalidImage := "rep@sha256:0b3787ac21ffb4edbd6710e0e60f991d5ded8d8a4f558209ef5987f73db4211a"
	_, _, _, err := splitImageName(invalidImage, "[:]", "format")

	assert.Error(t, err, "Expected error splitting image name")
}

func setupTestController(t *testing.T) (*mock_ecr.MockClient, *mock_docker.MockClient, *mock_sts.MockClient, *mock_tagging.MockClient) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockECR := mock_ecr.NewMockClient(ctrl)
	mockDocker := mock_docker.NewMockClient(ctrl)
	mockSTS := mock_sts.NewMockClient(ctrl)
	mockTagging := mock_tagging.NewMockClient(ctrl)

	return mockECR, mockDocker, mockSTS, mockTagging
}

func setupEnvironmentVar() {
	os.Setenv("AWS_ACCESS_KEY", "AKIDEXAMPLE")
	os.Setenv("AWS_SECRET_KEY", "secret")
	defer func() {
		os.Unsetenv("AWS_ACCESS_KEY")
		os.Unsetenv("AWS_SECRET_KEY")
	}()
}

func setAllPushImageFlags() *cli.Context {
	flagSet := flag.NewFlagSet("ecs-cli-push", 0)
	flagSet.Parse([]string{image})
	return cli.NewContext(nil, flagSet, nil)
}

func setAllPullImageFlags() *cli.Context {
	flagSet := flag.NewFlagSet("ecs-cli-pull", 0)
	flagSet.Parse([]string{image})
	return cli.NewContext(nil, flagSet, nil)
}
