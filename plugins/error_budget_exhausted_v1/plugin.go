package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"

	pluginslov1 "github.com/slok/sloth/pkg/prometheus/plugin/slo/v1"
)

const (
	PluginVersion = "prometheus/slo/v1"
	// TODO - correct pluginID?
	PluginID = "github.com/wbollock/sloth-plugins/error_budget_exhausted_alert/v1"
)

// This plugin is intended to add an alert when an error budget is exhausted
// Useful to initiate organization policies around error budget depletion
// More informational than directly actionable as other burn alerts should fire first

type Config struct {
	Threshold   float64           `json:"threshold"`            // default 0, fully exhausted
	For         string            `json:"for"`                  // default 5m
	AlertName   string            `json:"alert_name,omitempty"` // default "ErrorBudgetExhausted"
	AlertLabels map[string]string `json:"alert_labels,omitempty"`
}

func NewPlugin(configData json.RawMessage, _ pluginslov1.AppUtils) (pluginslov1.Plugin, error) {
	cfg := Config{
		// defaults
		Threshold:   0,
		For:         "5m",
		AlertName:   "ErrorBudgetExhausted",
		AlertLabels: map[string]string{},
	}
	if err := json.Unmarshal(configData, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return plugin{config: cfg}, nil
}

type plugin struct {
	config Config
}

type PrometheusRule struct {
	Alert       string            `yaml:"alert"`
	Expr        string            `yaml:"expr"`
	For         model.Duration    `yaml:"for,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

type PrometheusRuleGroup struct {
	Name     string           `yaml:"name"`
	Interval string           `yaml:"interval,omitempty"`
	Rules    []PrometheusRule `yaml:"rules"`
}

// labelMatcher takes a map of labels and returns a string for PromQL inclusion
func labelMatcher(labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf(`%s="%s"`, k, labels[k]))
	}
	return strings.Join(parts, ",")
}

func (p plugin) ProcessSLO(_ context.Context, req *pluginslov1.Request, result *pluginslov1.Result) error {
	slo := &req.SLO

	labels := map[string]string{
		"sloth_slo":     slo.Name,
		"sloth_service": slo.Service,
		"sloth_id":      fmt.Sprintf("%s-%s", slo.Service, slo.Name),
	}

	for k, v := range p.config.AlertLabels {
		labels[k] = v
	}

	// this is the right time series right?
	// these labels should be different than alert labels..
	// TODO - should be able to get labels from the SLO itself like normal alerts, ideally
	// TODO - need function around this for gauge
	expr := fmt.Sprintf(`slo:period_error_budget_remaining:ratio{%s} <= 0`, labelMatcher(labels))

	// Convert the 'for' field to model.Duration type
	// TODO - best place to do this?
	forDuration, err := model.ParseDuration(p.config.For)
	if err != nil {
		return fmt.Errorf("invalid 'for' duration: %w", err)
	}

	result.SLORules.AlertRules.Rules = append(result.SLORules.AlertRules.Rules, rulefmt.Rule{
		Alert:  fmt.Sprintf("%s", p.config.AlertName),
		Expr:   expr,
		For:    forDuration,
		Labels: p.config.AlertLabels,
		Annotations: map[string]string{
			"description": fmt.Sprintf("Error budget exhausted for SLO: %s", slo.Name),
		},
	})

	return nil
}
