# Safe AWS Serverless website deployments made easy

The [AWS Serverless Application Model][sam] (SAM) defines a simplified framework
on top of CloudFormation for building serverless apps. It has great support for
[safe deployments][safe-doc] and is well-documented. This safe deployment functionality
works for a Lambda in _any_ scenario - not just websites. As such, it's very
general-purpose. This flexibility comes at the cost of having to reinvent a
pretty common wheel. That's where this app comes in.

## How it works

When configured as a `PreTraffic` "hook" for your `AWS::Serverless::Function`, 
the following happens during a `sam deploy`:

* New _version_ of Lambda function is published. No traffic is going to it
  yet. The function's _alias_ is still sending 100% of traffic to the previous
  version. 

* CloudFormation notifies CodeDeploy that there is a new version of the function
  ready to start receiving traffic.
  
* CodeDeploy invokes the pre-traffic hook (**this** app) and tells it that
  it's ready to start the traffic shifting. At this point no traffic has been
  shifted yet.
  
* This app interprets the CodeDeploy hook payload to figure out which version
  of your function has just been deployed. It then invokes the new function 
  version with a `GET /healthcheck` request in API Gateway* format.
  
* If the new code version return a 2xx-3xx: success! This app notifies CodeDeploy
  that the new function version is good to be deployed.
  
* CodeDeploy now starts shifting traffic to the new version at the rate defined
  in your `DeploymentPreference.Type` property.
  
* Alternatively, if the new function code did **not** return a 2xx or 3xx, it is
  considered non-viable and this app notifies CodeDeploy that it's no good.
  
* CodeDeploy marks the deployment as failed, reports back to CloudFormation and
  the stack is rolled back.

## Installation

The app is installable through the [Serverless App Repository][sar]. It's named
`codedeployhook` and ARN is `arn:aws:serverlessrepo:us-east-1:607481581596:applications/codedeployhook`.

## Usage

You should modify your `AWS::Serverless::Function` resources to look like this
(if they're a website!):

```yaml
  Function:
    Type: AWS::Serverless::Function
    Properties:
      Handler: index.handler
      Runtime: python3.7
      Events:
        Api:
          Type: Api
      # now add the following
      AutoPublishAlias: live
      DeploymentPreference:
        Type: AllAtOnce
        Role: !ImportValue ServerlessDeploymentRoleArn
        Hooks:
          PreTraffic: !ImportValue PreTrafficHook      
```

## Roadmap

* Right now, the app assumes your serverless template only has a single function
  with a `DeploymentPreference` configured. If you have multiple functions in a
  single template, only the first will be checked. This will change to check all 
  configured functions concurrently.
 
[sam]: https://github.com/awslabs/serverless-application-model
[safe-doc]: https://github.com/awslabs/serverless-application-model/blob/master/docs/safe_lambda_deployments.rst#traffic-shifting-using-codedeploy
[sar]: https://console.aws.amazon.com/lambda/home?region=us-east-1#/create/app?applicationId=arn:aws:serverlessrepo:us-east-1:607481581596:applications/sam-alb
