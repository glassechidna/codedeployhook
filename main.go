package main

import (
	"codedeployhook/pkg/codedeployhook"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/codedeploy"
	lambdaapi "github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/sts"
	"os"
)

func main() {
	sess := session.Must(session.NewSession())

	stsapi := sts.New(sess)
	resp, err := stsapi.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		panic(err)
	}

	codedeployhook.AwsAccountId = *resp.Account
	codedeployhook.AwsRegion = os.Getenv("AWS_REGION")

	h := codedeployhook.NewHandler(codedeploy.New(sess), lambdaapi.New(sess))
	setenv(&h.Method, "LH_METHOD")
	setenv(&h.Path, "LH_PATH")
	setenv(&h.Host, "LH_HOST")

	lambda.Start(h.Handle)
}

func setenv(target *string, name string) {
	if val, found := os.LookupEnv(name); found {
		*target = val
	}
}
