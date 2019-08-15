package codedeployhook

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/codedeploy"
	"github.com/aws/aws-sdk-go/service/codedeploy/codedeployiface"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

type mockCodeDeploy struct {
	mock.Mock
	codedeployiface.CodeDeployAPI
}

func (m *mockCodeDeploy) GetDeploymentWithContext(ctx aws.Context, input *codedeploy.GetDeploymentInput, opts ...request.Option) (*codedeploy.GetDeploymentOutput, error) {
	f := m.Called(ctx, input, opts)
	output, _ := f.Get(0).(*codedeploy.GetDeploymentOutput)
	return output, f.Error(1)
}

func (m *mockCodeDeploy) GetApplicationRevisionWithContext(ctx aws.Context, input *codedeploy.GetApplicationRevisionInput, opts ...request.Option) (*codedeploy.GetApplicationRevisionOutput, error) {
	f := m.Called(ctx, input, opts)
	output, _ := f.Get(0).(*codedeploy.GetApplicationRevisionOutput)
	return output, f.Error(1)
}

func (m *mockCodeDeploy) PutLifecycleEventHookExecutionStatusWithContext(ctx aws.Context, input *codedeploy.PutLifecycleEventHookExecutionStatusInput, opts ...request.Option) (*codedeploy.PutLifecycleEventHookExecutionStatusOutput, error) {
	f := m.Called(ctx, input, opts)
	output, _ := f.Get(0).(*codedeploy.PutLifecycleEventHookExecutionStatusOutput)
	return output, f.Error(1)
}

type mockLambda struct {
	mock.Mock
	lambdaiface.LambdaAPI
}

func (m *mockLambda) InvokeWithContext(ctx aws.Context, input *lambda.InvokeInput, opts ...request.Option) (*lambda.InvokeOutput, error) {
	f := m.Called(ctx, input, opts)
	output, _ := f.Get(0).(*lambda.InvokeOutput)
	return output, f.Error(1)
}

func setupMockCodeDeploy() *mockCodeDeploy {
	cd := &mockCodeDeploy{}

	cd.
		On("GetDeploymentWithContext", mock.Anything, mock.AnythingOfType("*codedeploy.GetDeploymentInput"), mock.AnythingOfType("[]request.Option")).
		Return(&codedeploy.GetDeploymentOutput{
			DeploymentInfo: &codedeploy.DeploymentInfo{
				ApplicationName: aws.String(""),
				Revision: &codedeploy.RevisionLocation{
					RevisionType: aws.String("String"),
				},
			},
		}, nil)

	cd.
		On("GetApplicationRevisionWithContext", mock.Anything, mock.AnythingOfType("*codedeploy.GetApplicationRevisionInput"), mock.AnythingOfType("[]request.Option")).
		Return(&codedeploy.GetApplicationRevisionOutput{
			ApplicationName: aws.String(""),
			Revision: &codedeploy.RevisionLocation{
				RevisionType: aws.String("String"),
				String_: &codedeploy.RawString{
					Content: aws.String(`
						{
						  "version": "0.0",
						  "Resources": [
							{
							  "test-stack-packaged-Function-UMKRBSJSONDN": {
								"Type": "AWS::Lambda::Function",
								"Properties": {
								  "Name": "test-stack-packaged-Function-UMKRBSJSONDN",
								  "Alias": "live",
								  "CurrentVersion": "2",
								  "TargetVersion": "3"
								}
							  }
							}
						  ],
						  "Hooks": [
							{
							  "BeforeAllowTraffic": "CodeDeployHook_abc"
							}
						  ]
						}
					`),
				},
			},
		}, nil)

	return cd
}

func TestHandler_Handle_FuncReturns200(t *testing.T) {
	cd := setupMockCodeDeploy()
	l := &mockLambda{}
	h := NewHandler(cd, l)

	l.
		On("InvokeWithContext", mock.Anything, mock.AnythingOfType("*lambda.InvokeInput"), mock.AnythingOfType("[]request.Option")).
		Return(&lambda.InvokeOutput{
			StatusCode: aws.Int64(200),
			Payload: []byte(`
				{
					"isBase64Encoded": false,
					"statusCode": 200,
					"statusDescription": "200 OK",
					"headers": {
						"Content-Type": "text/plain"
					},
					"body": "Hello from Lambda (optional)"
				}
			`),
			FunctionError: nil,
		}, nil)

	cd.
		On("PutLifecycleEventHookExecutionStatusWithContext", mock.Anything, &codedeploy.PutLifecycleEventHookExecutionStatusInput{
			Status:                        aws.String("Succeeded"),
			DeploymentId:                  aws.String("d-0123ABC"),
			LifecycleEventHookExecutionId: aws.String("someBase64Data"),
		}, mock.AnythingOfType("[]request.Option")).
		Return(&codedeploy.PutLifecycleEventHookExecutionStatusOutput{}, nil)

	assert.NotPanics(t, func() {
		err := h.Handle(context.Background(), CodeDeployEvent{
			DeploymentId:                  "d-0123ABC",
			LifecycleEventHookExecutionId: "someBase64Data",
		})
		assert.NoError(t, err)
	})

	cd.AssertExpectations(t)
	l.AssertExpectations(t)
}

func TestHandler_Handle_FuncReturns500(t *testing.T) {
	cd := setupMockCodeDeploy()
	l := &mockLambda{}
	h := NewHandler(cd, l)

	l.
		On("InvokeWithContext", mock.Anything, mock.Anything, mock.AnythingOfType("[]request.Option")).
		Run(func(args mock.Arguments) {
			input := args.Get(1).(*lambda.InvokeInput)
			assert.Equal(t, "arn:aws:lambda:unknown:unknown:function:test-stack-packaged-Function-UMKRBSJSONDN:3", *input.FunctionName)
			assert.JSONEq(t, `
				{
				  "httpMethod": "GET",
				  "path": "/healthcheck",
				  "headers": {
					"Host": "example.com"
				  },
				  "multiValueHeaders": {
					"Host": [
					  "example.com"
					]
				  },
				  "requestContext": {
					"elb": {
					  "targetGroupArn": ""
					}
				  },
				  "isBase64Encoded": false,
				  "body": ""
				}
			`, string(input.Payload))
		}).
		Return(&lambda.InvokeOutput{
			StatusCode: aws.Int64(200),
			Payload: []byte(`
				{
					"isBase64Encoded": false,
					"statusCode": 500,
					"statusDescription": "500 Internal Server Error",
					"headers": {
						"Content-Type": "text/plain"
					},
					"body": "Hello from Lambda (optional)"
				}
			`),
			FunctionError: nil,
		}, nil)

	cd.
		On("PutLifecycleEventHookExecutionStatusWithContext", mock.Anything, &codedeploy.PutLifecycleEventHookExecutionStatusInput{
			Status:                        aws.String("Failed"),
			DeploymentId:                  aws.String("d-0123ABC"),
			LifecycleEventHookExecutionId: aws.String("someBase64Data"),
		}, mock.AnythingOfType("[]request.Option")).
		Return(&codedeploy.PutLifecycleEventHookExecutionStatusOutput{}, nil)

	assert.NotPanics(t, func() {
		err := h.Handle(context.Background(), CodeDeployEvent{
			DeploymentId:                  "d-0123ABC",
			LifecycleEventHookExecutionId: "someBase64Data",
		})
		assert.NoError(t, err)
	})

	cd.AssertExpectations(t)
	l.AssertExpectations(t)
}
