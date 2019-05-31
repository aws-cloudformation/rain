package client

import (
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws/awserr"
)

type Error error

func NewError(err error) Error {
	if err == nil {
		return nil
	}

	if err, ok := err.(awserr.Error); ok {
		return Error(errors.New(err.Message()))
	}

	return Error(err)
}
