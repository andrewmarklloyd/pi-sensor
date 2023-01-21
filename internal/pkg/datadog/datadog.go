package datadog

import (
	"context"
	"fmt"
	"time"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
)

type Client struct {
	api        *datadogV2.MetricsApi
	OPTokenExp time.Time
}

func NewDatadogClient() Client {
	configuration := datadog.NewConfiguration()
	apiClient := datadog.NewAPIClient(configuration)
	api := datadogV2.NewMetricsApi(apiClient)

	return Client{
		api: api,
	}
}

func (c *Client) PublishTokenDaysLeft(ctx context.Context, tokenExp, tokenName string) error {
	expTime, err := time.Parse(time.RFC3339, tokenExp)
	if err != nil {
		return fmt.Errorf("parsing time from token expiration: %w", err)
	}

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
						Name: datadog.PtrString(tokenName),
					},
					{
						Type: datadog.PtrString("owner"),
						Name: datadog.PtrString("opconnect"),
					},
				},
			},
		},
	}

	_, _, err = c.api.SubmitMetrics(ctx, body, *datadogV2.NewSubmitMetricsOptionalParameters())
	if err != nil {
		return fmt.Errorf("submitting metrics: %w", err)
	}

	return nil
}
