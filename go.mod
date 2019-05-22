module github.com/aws-cloudformation/rain

replace github.com/aws-cloudformation/rain => ./

require (
	github.com/aws/aws-sdk-go-v2 v0.7.0
	github.com/awslabs/aws-cloudformation-template-formatter v0.5.0
	github.com/spf13/cobra v0.0.4
	gopkg.in/yaml.v2 v2.2.2
)
