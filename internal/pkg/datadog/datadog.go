package datadog

import (
	"context"
	"fmt"
	"time"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/config"
)

type Client struct {
	api    *datadogV2.MetricsApi
	apiKey string
	appKey string
}

func NewDatadogClient(apiKey, appKey string) Client {
	configuration := datadog.NewConfiguration()
	apiClient := datadog.NewAPIClient(configuration)
	api := datadogV2.NewMetricsApi(apiClient)

	return Client{
		api:    api,
		apiKey: apiKey,
		appKey: appKey,
	}
}

func (c *Client) PublishTokenDaysLeft(ctx context.Context, tokenMetadata config.TokenMetadata) error {
	expTime, err := time.Parse(time.RFC3339, tokenMetadata.Expiration)
	if err != nil {
		return fmt.Errorf("parsing time from token expiration: %w", err)
	}

	valueCtx := context.WithValue(
		ctx,
		datadog.ContextAPIKeys,
		map[string]datadog.APIKey{
			"apiKeyAuth": {
				Key: c.apiKey,
			},
			"appKeyAuth": {
				Key: c.appKey,
			},
		},
	)

	body := datadogV2.MetricPayload{
		Series: []datadogV2.MetricSeries{
			{
				Metric: "token.days_left",
				Type:   datadogV2.METRICINTAKETYPE_COUNT.Ptr(),
				Points: []datadogV2.MetricPoint{
					{
						Timestamp: datadog.PtrInt64(time.Now().Unix()),
						Value:     datadog.PtrFloat64(time.Until(expTime).Hours() / 24),
					},
				},
				Resources: []datadogV2.MetricResource{
					{
						Type: datadog.PtrString("tokenname"),
						Name: datadog.PtrString(tokenMetadata.Name),
					},
					{
						Type: datadog.PtrString("owner"),
						Name: datadog.PtrString(tokenMetadata.Owner),
					},
				},
			},
		},
	}

	_, _, err = c.api.SubmitMetrics(valueCtx, body, *datadogV2.NewSubmitMetricsOptionalParameters())
	if err != nil {
		return fmt.Errorf("submitting metrics: %s", err)
	}

	return nil
}

func (c *Client) PublishHeartbeat(ctx context.Context, agent string) error {
	valueCtx := context.WithValue(
		ctx,
		datadog.ContextAPIKeys,
		map[string]datadog.APIKey{
			"apiKeyAuth": {
				Key: c.apiKey,
			},
			"appKeyAuth": {
				Key: c.appKey,
			},
		},
	)

	body := datadogV2.MetricPayload{
		Series: []datadogV2.MetricSeries{
			{
				Metric: "agent.heartbeat",
				Type:   datadogV2.METRICINTAKETYPE_COUNT.Ptr(),
				Points: []datadogV2.MetricPoint{
					{
						Timestamp: datadog.PtrInt64(time.Now().Unix()),
						Value:     datadog.PtrFloat64(1),
					},
				},
				Resources: []datadogV2.MetricResource{
					{
						Type: datadog.PtrString("source"),
						Name: datadog.PtrString("pi-sensor"),
					},
					{
						Type: datadog.PtrString("service"),
						Name: datadog.PtrString("pi-sensor"),
					},
					{
						Type: datadog.PtrString("agent"),
						Name: datadog.PtrString(agent),
					},
				},
			},
		},
	}

	_, _, err := c.api.SubmitMetrics(valueCtx, body, *datadogV2.NewSubmitMetricsOptionalParameters())
	if err != nil {
		return fmt.Errorf("submitting metrics: %s", err)
	}

	return nil
}
