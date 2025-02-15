// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2021 Datadog, Inc.

package datadogclient

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/go-logr/logr"

	"github.com/DataDog/datadog-operator/pkg/config"

	datadogapi "github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	datadogV1 "github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
	apicommon "github.com/DataDog/datadog-operator/apis/datadoghq/common"
)

const prefix = "https://api."

// DatadogMonitorClient contains the Datadog Monitor API Client and Authentication context.
type DatadogMonitorClient struct {
	Client *datadogV1.MonitorsApi
	Auth   context.Context
}

// InitDatadogMonitorClient initializes the Datadog Monitor API Client and establishes credentials.
func InitDatadogMonitorClient(logger logr.Logger, creds config.Creds) (DatadogMonitorClient, error) {
	if creds.APIKey == "" || creds.AppKey == "" {
		return DatadogMonitorClient{}, errors.New("error obtaining API key and/or app key")
	}

	// Initialize the official Datadog V1 API client.
	authV1 := context.WithValue(
		context.Background(),
		datadogapi.ContextAPIKeys,
		map[string]datadogapi.APIKey{
			"apiKeyAuth": {
				Key: creds.APIKey,
			},
			"appKeyAuth": {
				Key: creds.AppKey,
			},
		},
	)
	configV1 := datadogapi.NewConfiguration()

	apiURL := ""
	if os.Getenv(config.DDURLEnvVar) != "" {
		apiURL = os.Getenv(config.DDURLEnvVar)
	} else if site := os.Getenv(apicommon.DDSite); site != "" {
		apiURL = prefix + strings.TrimSpace(site)
	}

	if apiURL != "" {
		logger.Info("Got API URL for DatadogOperator controller", "URL", apiURL)
		parsedAPIURL, parseErr := url.Parse(apiURL)
		if parseErr != nil {
			return DatadogMonitorClient{}, fmt.Errorf(`invalid API URL : %w`, parseErr)
		}
		if parsedAPIURL.Host == "" || parsedAPIURL.Scheme == "" {
			return DatadogMonitorClient{}, fmt.Errorf(`missing protocol or host : %s`, apiURL)
		}
		// If API URL is passed, set and use the API name and protocol on ServerIndex{1}.
		authV1 = context.WithValue(authV1, datadogapi.ContextServerIndex, 1)
		authV1 = context.WithValue(authV1, datadogapi.ContextServerVariables, map[string]string{
			"name":     parsedAPIURL.Host,
			"protocol": parsedAPIURL.Scheme,
		})
	}

	apiClient := datadogapi.NewAPIClient(configV1)
	client := datadogV1.NewMonitorsApi(apiClient)

	return DatadogMonitorClient{Client: client, Auth: authV1}, nil
}
