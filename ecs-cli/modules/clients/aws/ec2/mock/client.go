// Copyright 2015-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

// Automatically generated by MockGen. DO NOT EDIT!
// Source: github.com/aws/amazon-ecs-cli/ecs-cli/modules/clients/aws/ec2 (interfaces: EC2Client)

package mock_ec2

import (
	ec2 "github.com/aws/aws-sdk-go/service/ec2"
	gomock "github.com/golang/mock/gomock"
)

// Mock of EC2Client interface
type MockEC2Client struct {
	ctrl     *gomock.Controller
	recorder *_MockEC2ClientRecorder
}

// Recorder for MockEC2Client (not exported)
type _MockEC2ClientRecorder struct {
	mock *MockEC2Client
}

func NewMockEC2Client(ctrl *gomock.Controller) *MockEC2Client {
	mock := &MockEC2Client{ctrl: ctrl}
	mock.recorder = &_MockEC2ClientRecorder{mock}
	return mock
}

func (_m *MockEC2Client) EXPECT() *_MockEC2ClientRecorder {
	return _m.recorder
}

func (_m *MockEC2Client) DescribeInstances(_param0 []*string) (map[string]*ec2.Instance, error) {
	ret := _m.ctrl.Call(_m, "DescribeInstances", _param0)
	ret0, _ := ret[0].(map[string]*ec2.Instance)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockEC2ClientRecorder) DescribeInstances(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "DescribeInstances", arg0)
}

func (_m *MockEC2Client) DescribeNetworkInterfaces(_param0 []*string) ([]*ec2.NetworkInterface, error) {
	ret := _m.ctrl.Call(_m, "DescribeNetworkInterfaces", _param0)
	ret0, _ := ret[0].([]*ec2.NetworkInterface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockEC2ClientRecorder) DescribeNetworkInterfaces(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "DescribeNetworkInterfaces", arg0)
}
