module github.com/aws-cloudformation/rain

go 1.22.1

toolchain go1.22.4

require (
	//github.com/ake-persson/mapslice-json v0.0.0-20210720081907-22c8edf57807
	github.com/appscode/jsonpatch v1.0.1
	github.com/aws/aws-sdk-go-v2 v1.32.6
	github.com/aws/aws-sdk-go-v2/config v1.28.6
	github.com/aws/aws-sdk-go-v2/credentials v1.17.47
	github.com/aws/aws-sdk-go-v2/service/bedrockruntime v1.23.0
	github.com/aws/aws-sdk-go-v2/service/cloudformation v1.56.1
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.198.0
	github.com/aws/aws-sdk-go-v2/service/s3 v1.71.0
	github.com/aws/aws-sdk-go-v2/service/sts v1.33.2
	github.com/aws/smithy-go v1.22.1
	github.com/chzyer/readline v1.5.1
	github.com/google/go-cmp v0.6.0
	github.com/google/uuid v1.6.0
	github.com/gookit/color v1.5.4
	github.com/nathan-fiscaletti/consolesize-go v0.0.0-20220204101620-317176b6684d
	github.com/spf13/cobra v1.8.1
	github.com/spf13/pflag v1.0.5
	golang.org/x/sys v0.28.0
	golang.org/x/term v0.27.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/apple/pkl-go v0.8.0
	github.com/aws/aws-lambda-go v1.47.0
	github.com/aws/aws-sdk-go-v2/service/acm v1.30.7
	github.com/aws/aws-sdk-go-v2/service/cloudfront v1.43.1
	github.com/aws/aws-sdk-go-v2/service/codeartifact v1.33.7
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.38.0
	github.com/aws/aws-sdk-go-v2/service/kms v1.37.7
	github.com/aws/aws-sdk-go-v2/service/lightsail v1.42.7
	github.com/aws/aws-sdk-go-v2/service/rds v1.93.0
	github.com/aws/aws-sdk-go-v2/service/sagemaker v1.169.0
	github.com/aws/aws-sdk-go-v2/service/servicequotas v1.25.7
	github.com/aws/aws-sdk-go-v2/service/ssm v1.56.1
	github.com/fatih/color v1.18.0
	github.com/gabriel-vasile/mimetype v1.4.7
	github.com/lestrrat-go/jwx/v2 v2.1.3
	github.com/manifoldco/promptui v0.9.0
	github.com/mattn/go-runewidth v0.0.15
	github.com/stretchr/testify v1.9.0
)

require (
	github.com/aws/aws-sdk-go v1.55.5 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.25 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.10.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.28.6 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.3.0 // indirect
	github.com/goccy/go-json v0.10.4 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/lestrrat-go/blackmagic v1.0.2 // indirect
	github.com/lestrrat-go/httpcc v1.0.1 // indirect
	github.com/lestrrat-go/httprc v1.0.6 // indirect
	github.com/lestrrat-go/iter v1.0.2 // indirect
	github.com/lestrrat-go/option v1.0.1 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/segmentio/asm v1.2.0 // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	golang.org/x/crypto v0.31.0 // indirect
	golang.org/x/net v0.32.0 // indirect
)

require (
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.7 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.25 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.25 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/cloudcontrol v1.23.2
	github.com/aws/aws-sdk-go-v2/service/iam v1.38.2
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.4.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.18.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.24.7 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.6 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	golang.org/x/exp v0.0.0-20241217172543-b2144cdd0a67
	gopkg.in/yaml.v2 v2.4.0
)
