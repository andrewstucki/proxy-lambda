package main

import (
	_ "embed"

	"github.com/aws/aws-lambda-go/lambda"

	"github.com/andrewstucki/proxy-lambda/proxy"
)

//go:embed config.json
var config []byte

func main() {
	lambda.Start(proxy.Run(config))
}
