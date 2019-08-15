package codedeployhook

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/codedeploy"
	"github.com/aws/aws-sdk-go/service/codedeploy/codedeployiface"
	lambdaapi "github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
	"github.com/pkg/errors"
)

type Handler struct {
	codedeploy codedeployiface.CodeDeployAPI
	lambda     lambdaiface.LambdaAPI
}

func NewHandler(codedeploy codedeployiface.CodeDeployAPI, lambda lambdaiface.LambdaAPI) *Handler {
	return &Handler{codedeploy: codedeploy, lambda: lambda}
}

func (h *Handler) Handle(ctx context.Context, event CodeDeployEvent) (err error) {
	defer func() {
		if rerr := recover(); rerr != nil {
			err = errors.Errorf("panic: %s", rerr)
		}
		if err != nil {
			h.codedeploy.PutLifecycleEventHookExecutionStatusWithContext(ctx, &codedeploy.PutLifecycleEventHookExecutionStatusInput{
				DeploymentId:                  &event.DeploymentId,
				LifecycleEventHookExecutionId: &event.LifecycleEventHookExecutionId,
				Status:                        aws.String("Failed"),
			})
		}
	}()
	deployInfo, err := h.codedeploy.GetDeploymentWithContext(ctx, &codedeploy.GetDeploymentInput{DeploymentId: &event.DeploymentId})
	if err != nil {
		return errors.WithStack(err)
	}

	revisionInfoResp, err := h.codedeploy.GetApplicationRevisionWithContext(ctx, &codedeploy.GetApplicationRevisionInput{
		ApplicationName: deployInfo.DeploymentInfo.ApplicationName,
		Revision:        deployInfo.DeploymentInfo.Revision,
	})
	if err != nil {
		return errors.WithStack(err)
	}

	info := RevisionInfo{}
	err = json.Unmarshal([]byte(*revisionInfoResp.Revision.String_.Content), &info)
	if err != nil {
		return errors.WithStack(err)
	}

	arn := info.functionArn()

	bytes, err := json.Marshal(events.ALBTargetGroupRequest{
		HTTPMethod:        "GET",
		Path:              "/healthcheck",
		Headers:           map[string]string{"Host": "example.com"},
		MultiValueHeaders: map[string][]string{"Host": {"example.com"}},
	})
	if err != nil {
		return errors.WithStack(err)
	}

	invokeResp, err := h.lambda.InvokeWithContext(ctx, &lambdaapi.InvokeInput{
		FunctionName: &arn,
		Payload:      bytes,
	})
	if err != nil {
		return errors.WithStack(err)
	}

	status := "Failed"
	if invocationWasSuccessful(invokeResp) {
		status = "Succeeded"
	}

	_, err = h.codedeploy.PutLifecycleEventHookExecutionStatusWithContext(ctx, &codedeploy.PutLifecycleEventHookExecutionStatusInput{
		DeploymentId:                  &event.DeploymentId,
		LifecycleEventHookExecutionId: &event.LifecycleEventHookExecutionId,
		Status:                        &status,
	})
	return errors.WithStack(err)
}

func invocationWasSuccessful(response *lambdaapi.InvokeOutput) bool {
	if response.FunctionError != nil && len(*response.FunctionError) > 0 {
		return false
	}

	if *response.StatusCode != 200 {
		return false
	}

	payload := events.ALBTargetGroupResponse{}
	err := json.Unmarshal(response.Payload, &payload)
	if err != nil {
		return false
	}

	if payload.StatusCode < 200 || payload.StatusCode > 399 {
		return false
	}

	return true
}
