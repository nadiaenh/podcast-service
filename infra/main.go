package main

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/lambda"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/s3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := config.New(ctx, "")
		apiKey := cfg.RequireSecret("apiKey")
		anthropicKey := cfg.RequireSecret("anthropicApiKey")
		elevenlabsKey := cfg.GetSecret("elevenlabsApiKey").ToStringOutput()
		elevenlabsVoice := cfg.Get("elevenlabsVoiceId")
		voxtralKey := cfg.GetSecret("voxtralApiKey").ToStringOutput()
		voxtralVoice := cfg.Get("voxtralVoiceId")
		ttsProvider := cfg.Get("ttsProvider")
		if ttsProvider == "" {
			ttsProvider = "elevenlabs"
		}

		bucket, err := s3.NewBucketV2(ctx, "mp3s", &s3.BucketV2Args{
			ForceDestroy: pulumi.Bool(true),
		})
		if err != nil {
			return err
		}

		role, err := iam.NewRole(ctx, "podcast-lambda-role", &iam.RoleArgs{
			AssumeRolePolicy: pulumi.String(`{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Principal": { "Service": "lambda.amazonaws.com" },
    "Action": "sts:AssumeRole"
  }]
}`),
		})
		if err != nil {
			return err
		}

		if _, err := iam.NewRolePolicyAttachment(ctx, "podcast-lambda-logs", &iam.RolePolicyAttachmentArgs{
			Role:      role.Name,
			PolicyArn: pulumi.String(iam.ManagedPolicyAWSLambdaBasicExecutionRole),
		}); err != nil {
			return err
		}

		if _, err := iam.NewRolePolicy(ctx, "podcast-lambda-s3", &iam.RolePolicyArgs{
			Role: role.ID(),
			Policy: bucket.Arn.ApplyT(func(arn string) (string, error) {
				return fmt.Sprintf(`{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": ["s3:PutObject", "s3:GetObject"],
    "Resource": "%s/*"
  }]
}`, arn), nil
			}).(pulumi.StringOutput),
		}); err != nil {
			return err
		}

		envVars := pulumi.StringMap{
			"API_KEY":             apiKey,
			"ANTHROPIC_API_KEY":   anthropicKey,
			"TTS_PROVIDER":        pulumi.String(ttsProvider),
			"S3_BUCKET":           bucket.Bucket,
			"ELEVENLABS_API_KEY":  elevenlabsKey,
			"ELEVENLABS_VOICE_ID": pulumi.String(elevenlabsVoice),
			"VOXTRAL_API_KEY":     voxtralKey,
			"VOXTRAL_VOICE_ID":    pulumi.String(voxtralVoice),
		}

		fn, err := lambda.NewFunction(ctx, "podcast-lambda", &lambda.FunctionArgs{
			Runtime: pulumi.String("provided.al2023"),
			Handler: pulumi.String("bootstrap"),
			Code:    pulumi.NewFileArchive("../lambda/bootstrap.zip"),
			Role:    role.Arn,
			Architectures: pulumi.StringArray{
				pulumi.String("arm64"),
			},
			Timeout:     pulumi.Int(120),
			MemorySize:  pulumi.Int(512),
			Environment: &lambda.FunctionEnvironmentArgs{Variables: envVars},
		})
		if err != nil {
			return err
		}

		fnUrl, err := lambda.NewFunctionUrl(ctx, "podcast-lambda-url", &lambda.FunctionUrlArgs{
			FunctionName:      fn.Name,
			AuthorizationType: pulumi.String("NONE"),
			Cors: &lambda.FunctionUrlCorsArgs{
				AllowMethods: pulumi.StringArray{pulumi.String("POST")},
				AllowOrigins: pulumi.StringArray{pulumi.String("*")},
				AllowHeaders: pulumi.StringArray{pulumi.String("content-type"), pulumi.String("x-api-key")},
			},
		})
		if err != nil {
			return err
		}

		if _, err := lambda.NewPermission(ctx, "podcast-lambda-url-public", &lambda.PermissionArgs{
			Action:              pulumi.String("lambda:InvokeFunctionUrl"),
			Function:            fn.Name,
			Principal:           pulumi.String("*"),
			FunctionUrlAuthType: pulumi.String("NONE"),
		}); err != nil {
			return err
		}

		if _, err := lambda.NewPermission(ctx, "podcast-lambda-invoke-public", &lambda.PermissionArgs{
			Action:    pulumi.String("lambda:InvokeFunction"),
			Function:  fn.Name,
			Principal: pulumi.String("*"),
		}); err != nil {
			return err
		}

		ctx.Export("functionUrl", fnUrl.FunctionUrl)
		return nil
	})
}
