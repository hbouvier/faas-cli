// Copyright (c) Alex Ellis 2017. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	types "github.com/openfaas/faas-provider/types"
)

// ListFunctions list deployed functions
func (c *Client) ListFunctions(ctx context.Context, namespace string) ([]types.FunctionStatus, error) {
	var (
		results      []types.FunctionStatus
		listEndpoint string
		err          error
	)

	c.AddCheckRedirect(func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	})

	listEndpoint = systemPath
	if len(namespace) > 0 {
		listEndpoint, err = addQueryParams(listEndpoint, map[string]string{namespaceKey: namespace})
		if err != nil {
			return results, err
		}
	}

	retries := 0
	done := false
	nextURL := listEndpoint
	for !done {
		if retries > 5 {
			return nil, fmt.Errorf("too many redirection (%d) to OpenFaaS on URL: %s", retries, c.GatewayURL.String())
		}
		retries += 1
		getRequest, err := c.newRequest(http.MethodGet, nextURL, nil)
		if err != nil {
			return nil, fmt.Errorf("cannot connect to OpenFaaS on URL: %s", c.GatewayURL.String())
		}

		res, err := c.doRequest(ctx, getRequest)
		if err != nil {
			return nil, fmt.Errorf("cannot connect to OpenFaaS on URL: %s", c.GatewayURL.String())
		}

		if res.Body != nil {
			defer res.Body.Close()
		}

		// fmt.Println("URL:", nextURL)
		// fmt.Println("Retry:", retries)
		// fmt.Println("StatusCode:", res.StatusCode)

		switch res.StatusCode {
		case http.StatusOK:
			done = true
			bytesOut, err := ioutil.ReadAll(res.Body)
			if err != nil {
				return nil, fmt.Errorf("cannot read result from OpenFaaS on URL: %s", c.GatewayURL.String())
			}
			jsonErr := json.Unmarshal(bytesOut, &results)
			if jsonErr != nil {
				return nil, fmt.Errorf("cannot parse result from OpenFaaS on URL: %s\n%s", c.GatewayURL.String(), jsonErr.Error())
			}
		case http.StatusUnauthorized:
			return nil, fmt.Errorf("unauthorized access, run \"faas-cli login\" to setup authentication for this server")
		case http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
			nextURL = res.Header.Get("Location")
		default:
			done = true
			bytesOut, err := ioutil.ReadAll(res.Body)
			if err == nil {
				return nil, fmt.Errorf("server returned unexpected status code: %d - %s", res.StatusCode, string(bytesOut))
			}
		}		
	}
	return results, nil
}
