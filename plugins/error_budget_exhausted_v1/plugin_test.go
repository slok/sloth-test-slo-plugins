package plugin_test

import (
	"encoding/json"
	"testing"
	"time"

	prommodel "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/stretchr/testify/assert"

	model "github.com/slok/sloth/pkg/common/model"
	pluginslov1 "github.com/slok/sloth/pkg/prometheus/plugin/slo/v1"
	pluginslov1testing "github.com/slok/sloth/pkg/prometheus/plugin/slo/v1/testing"

	plugin "github.com/wbollock/sloth-plugins/error_budget_exhausted_v1"
)

func MustJSONRawMessage(t *testing.T, v any) json.RawMessage {
	j, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}
	return j
}

func TestPlugin(t *testing.T) {
	tests := map[string]struct {
		config     json.RawMessage
		req        pluginslov1.Request
		res        pluginslov1.Result
		expRes     pluginslov1.Result
		expLoadErr bool
		expErr     bool
	}{
		"Should create a correct alert rule for exhausted error budget.": {
			config: MustJSONRawMessage(t, plugin.Config{
				Threshold: 0.0,
				For:       "5m",
				AlertName: "ErrorBudgetExhausted",
				AlertLabels: map[string]string{
					"severity": "critical",
					"team":     "platform",
				},
			}),
			req: pluginslov1.Request{
				SLO: model.PromSLO{
					Name:    "availability",
					Service: "checkout",
					// TODO - think about label strategy here
					// Labels: map[string]string{
					// 	"component":   "api",
					// 	"environment": "prod",
					// },
				},
			},
			// TODO - need more unhappy test cases and such
			res: pluginslov1.Result{},
			expRes: pluginslov1.Result{
				SLORules: model.PromSLORules{
					AlertRules: model.PromRuleGroup{
						Rules: []rulefmt.Rule{
							{
								Alert: "ErrorBudgetExhausted",
								Expr:  `slo:period_error_budget_remaining:ratio{severity="critical",sloth_id="checkout-availability",sloth_service="checkout",sloth_slo="availability",team="platform"} <= 0`,
								For:   prommodel.Duration(5 * time.Minute),
								Labels: map[string]string{
									"severity": "critical",
									"team":     "platform",
								},
								Annotations: map[string]string{
									"description": "Error budget exhausted for SLO: availability",
								},
							},
						},
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			p, err := pluginslov1testing.NewTestPlugin(t.Context(), pluginslov1testing.TestPluginConfig{
				PluginConfiguration: test.config,
			})
			if test.expLoadErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)

			err = p.ProcessSLO(t.Context(), &test.req, &test.res)
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expRes, test.res)
			}
		})
	}
}
