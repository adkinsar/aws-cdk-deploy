package main

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsapigateway"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsdynamodb"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/pipelines"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type GoCdkStackProps struct {
	awscdk.StackProps
}

func NewGoCdkApplication(scope constructs.Construct, id string, props *GoCdkStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	// create DynamoDB table
	table := awsdynamodb.NewTable(stack, jsii.String("myUserTable"), &awsdynamodb.TableProps{
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("username"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		TableName:     jsii.String("userTable"),
		RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
	})

	lambda := awslambda.NewFunction(stack, jsii.String("MyFunction"), &awslambda.FunctionProps{
		Runtime: awslambda.Runtime_PROVIDED_AL2023(),
		Handler: jsii.String("main"),
		Code:    awslambda.Code_FromAsset(jsii.String("lambda/function.zip"), nil),
	})

	table.GrantReadWriteData(lambda)

	api := awsapigateway.NewRestApi(stack, jsii.String("myWebAPI"), &awsapigateway.RestApiProps{
		DefaultCorsPreflightOptions: &awsapigateway.CorsOptions{
			AllowOrigins: awsapigateway.Cors_ALL_ORIGINS(),
			AllowMethods: awsapigateway.Cors_ALL_METHODS(),
			AllowHeaders: awsapigateway.Cors_DEFAULT_HEADERS(),
		},
		// DeployOptions: &awsapigateway.StageOptions{
		// 	LoggingLevel: awsapigateway.MethodLoggingLevel_INFO,
		// },
	})

	integration := awsapigateway.NewLambdaIntegration(lambda, nil)

	// Define the routes
	registerResource := api.Root().AddResource(jsii.String("register"), nil)
	registerResource.AddMethod(jsii.String("POST"), integration, nil)

	loginResource := api.Root().AddResource(jsii.String("login"), nil)
	loginResource.AddMethod(jsii.String("POST"), integration, nil)

	return stack
}

func NewGoCdkPipeline(scope constructs.Construct, id string, props *GoCdkStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	// Establish connection to GitHub repostory
	repo := pipelines.CodePipelineSource_Connection(jsii.String("adkinsar/aws-cdk-deploy"), jsii.String("main"), &pipelines.ConnectionSourceOptions{
		ConnectionArn: jsii.String("arn:aws:codestar-connections:us-east-2:590184108925:connection/302d5868-5e56-4752-ac01-72b083c65678"),
	})

	// Build Stage

	// Step - Dowload Source Code, Install dependencies, and synthesize CDK stacks- TODO figure out how to use a base image
	build := pipelines.NewCodeBuildStep(jsii.String("Synth"), &pipelines.CodeBuildStepProps{
		Input:           repo,
		InstallCommands: &[]*string{jsii.String("./install.sh")},
		Commands:        &[]*string{jsii.String("./build.sh")},
	})
	pipeline := pipelines.NewCodePipeline(stack, jsii.String("user-management-pipeline"), &pipelines.CodePipelineProps{
		PipelineName: jsii.String("user-management-api"),
		Synth:        build,
	})
	// Artifact from build
	cdkTemplates := build.PrimaryOutput()

	// Lint CloudFormation template - TODO

	// Create stage for deploying the application stack
	deploy := NewGoCdkPipelineDeployStage(stack, "Deploy", nil)
	// Post deployment steps
	manualApproval := pipelines.NewManualApprovalStep(jsii.String("Teardown Approval"), nil)
	environmentTeardown := pipelines.NewShellStep(jsii.String("Teardown Application"), &pipelines.ShellStepProps{
		Input:           cdkTemplates,
		InstallCommands: &[]*string{jsii.String("npm install -g aws-cdk")},
		Commands: &[]*string{
			jsii.String("cdk destroy -f --app assembly-UserManagementPipelineStack-Deploy"), // figure out reference
		},
	})
	environmentTeardown.AddStepDependency(manualApproval) // manual approval required to destroy stack

	// There is a better way to do this most likely
	// pipeline.Pipeline().Role().AddManagedPolicy(awsiam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("AdministratorAccess")))

	pipeline.AddStage(deploy, &pipelines.AddStageOpts{
		Post: &[]pipelines.Step{
			manualApproval,
			environmentTeardown,
		},
	})

	return stack
}

type GoCdkPipelineStage struct {
	awscdk.StageProps
}

// This deploys the actual cloud application infrastructure
func NewGoCdkPipelineDeployStage(scope constructs.Construct, id string, props *GoCdkPipelineStage) awscdk.Stage {
	var sprops awscdk.StageProps
	if props != nil {
		sprops = props.StageProps
	}
	stage := awscdk.NewStage(scope, &id, &sprops)

	NewGoCdkApplication(stage, "UserManagementStack", &GoCdkStackProps{
		awscdk.StackProps{
			Env: env(),
		},
	})

	return stage
}

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)

	NewGoCdkPipeline(app, "UserManagementPipelineStack", &GoCdkStackProps{
		awscdk.StackProps{
			Env: env(),
		},
	})

	app.Synth(nil)
}

// env determines the AWS environment (account+region) in which our stack is to
// be deployed. For more information see: https://docs.aws.amazon.com/cdk/latest/guide/environments.html
func env() *awscdk.Environment {
	// If unspecified, this stack will be "environment-agnostic".
	// Account/Region-dependent features and context lookups will not work, but a
	// single synthesized template can be deployed anywhere.
	//---------------------------------------------------------------------------
	return nil

	// Uncomment if you know exactly what account and region you want to deploy
	// the stack to. This is the recommendation for production stacks.
	//---------------------------------------------------------------------------
	// return &awscdk.Environment{
	//  Account: jsii.String("123456789012"),
	//  Region:  jsii.String("us-east-1"),
	// }

	// Uncomment to specialize this stack for the AWS Account and Region that are
	// implied by the current CLI configuration. This is recommended for dev
	// stacks.
	//---------------------------------------------------------------------------
	// return &awscdk.Environment{
	//  Account: jsii.String(os.Getenv("CDK_DEFAULT_ACCOUNT")),
	//  Region:  jsii.String(os.Getenv("CDK_DEFAULT_REGION")),
	// }
}
