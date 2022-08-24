package proxy

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/aws/aws-lambda-go/events"
)

type rule struct {
	PathMatch           *Regexp  `json:"pathMatch"`
	StripPrefix         string   `json:"stripPrefix"`
	Target              string   `json:"target"`
	RewriteRequestBody  bool     `json:"rewriteRequestBody"`
	KeepRequestHeaders  []string `json:"keepRequestHeaders"`
	DropResponseHeaders []string `json:"dropResponseHeaders"`
}

type proxy struct {
	SkipTLS bool   `json:"skipTLSVerification"`
	Host    string `json:"host"`
	Rules   []rule `json:"rules"`

	client *http.Client
}

var (
	globalProxy *proxy
	once        sync.Once
)

type lambdaHandler func(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

func Run(configData []byte) lambdaHandler {
	once.Do(func() {
		globalProxy = &proxy{}
		if err := json.Unmarshal(configData, &globalProxy); err != nil {
			panic(err)
		}
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: globalProxy.SkipTLS}

		globalProxy.client = &http.Client{
			Transport: transport,
		}
	})

	return globalProxy.handle
}

func (p *proxy) handle(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	path := "/" + request.PathParameters["proxy"]
	for _, rule := range p.Rules {
		if rule.PathMatch != nil && rule.PathMatch.MatchString(path) {
			req, err := p.eventToRequest(rule, path, request)

			if err != nil {
				return events.APIGatewayProxyResponse{}, err
			}
			resp, err := p.client.Do(req)
			if err != nil {
				return events.APIGatewayProxyResponse{}, err
			}
			return p.responseToEvent(rule, resp)
		}
	}
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusNotFound,
	}, nil
}

// based off of https://github.com/awslabs/aws-lambda-go-api-proxy/blob/2eab254086326d21dc5988df3cee8ff62beeb7ec/core/requestv2.go#L114
func (p *proxy) eventToRequest(target rule, path string, req events.APIGatewayProxyRequest) (*http.Request, error) {
	decodedBody := []byte(req.Body)
	if req.IsBase64Encoded {
		base64Body, err := base64.StdEncoding.DecodeString(req.Body)
		if err != nil {
			return nil, err
		}
		decodedBody = base64Body
	}

	if target.RewriteRequestBody {
		decodedBody = bytes.ReplaceAll(decodedBody, []byte(p.Host), []byte(target.Target))
	}

	if target.StripPrefix != "" && len(target.StripPrefix) > 1 {
		if strings.HasPrefix(path, target.StripPrefix) {
			path = strings.Replace(path, target.StripPrefix, "", 1)
		}
	}
	path = target.Target + path

	if len(req.QueryStringParameters) > 0 {
		values := url.Values{}
		for key, value := range req.QueryStringParameters {
			values.Add(key, value)
		}
		path += "?" + values.Encode()
	}

	httpRequest, err := http.NewRequest(
		strings.ToUpper(req.RequestContext.HTTPMethod),
		path,
		bytes.NewReader(decodedBody),
	)

	if err != nil {
		return nil, err
	}

	httpRequest.RemoteAddr = req.RequestContext.Identity.SourceIP

	for _, header := range target.KeepRequestHeaders {
		for headerKey, headerValue := range req.Headers {
			// ensure any origin headers are forwarded and re-written
			if strings.EqualFold(headerKey, "origin") {
				httpRequest.Header.Add(headerKey, strings.ReplaceAll(headerValue, p.Host, target.Target))
				continue
			}

			if strings.EqualFold(headerKey, header) {
				for _, val := range strings.Split(headerValue, ",") {
					httpRequest.Header.Add(headerKey, strings.Trim(val, " "))
				}
			}
		}
	}

	// null this out since we're doing a client request
	httpRequest.RequestURI = ""

	return httpRequest, nil
}

// based off of https://github.com/awslabs/aws-lambda-go-api-proxy/blob/825e2ce1153ecb1cdb06127fc0a344d3f7815742/core/responsev2.go#L85
func (p *proxy) responseToEvent(target rule, resp *http.Response) (events.APIGatewayProxyResponse, error) {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	// rewrite the response body content for any matching hosts
	body = bytes.ReplaceAll(body, []byte(target.Target), []byte(p.Host))

	var output string
	isBase64 := false
	if utf8.Valid(body) {
		output = string(body)
	} else {
		output = base64.StdEncoding.EncodeToString(body)
		isBase64 = true
	}

	headers := make(map[string]string)

OUTER:
	for headerKey, headerValue := range http.Header(resp.Header) {
		for _, header := range target.DropResponseHeaders {
			if strings.EqualFold(header, headerKey) {
				continue OUTER
			}
		}
		headers[headerKey] = strings.Join(headerValue, ",")
	}

	return events.APIGatewayProxyResponse{
		StatusCode:      resp.StatusCode,
		Headers:         headers,
		Body:            output,
		IsBase64Encoded: isBase64,
	}, nil
}
