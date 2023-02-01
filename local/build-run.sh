 #!/bin/bash
 go build ./cmd/rain

#  ./rain --debug --profile ezbeard-cep \
#    forecast test/templates/forecast.yml forecast22 \
#    --type AWS::S3::Bucket

./rain --debug --profile ezbeard-cep \
    deploy test/templates/success.template