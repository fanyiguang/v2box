package v2rayjson

import (
	"bytes"

	"github.com/sagernet/sing-box/common/json"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common"
	"github.com/sagernet/sing/common/format"
	"github.com/sagernet/sing/common/logger"

	v4json "github.com/v2fly/v2ray-core/v5/infra/conf/v4"
)

func Migrate(content []byte, logger logger.Logger) (option.Options, error) {
	var options option.Options
	var v2rayConfig v4json.Config
	decoder := json.NewDecoder(json.NewCommentFilter(bytes.NewReader(content)))
	err := decoder.Decode(&v2rayConfig)
	if err != nil {
		return option.Options{}, err
	}
	if logConfig := v2rayConfig.LogConfig; logConfig != nil && logConfig.ErrorLog != "" || logConfig.LogLevel != "" {
		options.Log = &option.LogOptions{}
		options.Log.Output = logConfig.ErrorLog
		options.Log.Level = logConfig.LogLevel
	}
	migrateDNS(common.PtrValueOrDefault(v2rayConfig.DNSConfig), &options)
	for i, inboundConfig := range v2rayConfig.InboundConfigs {
		inbound, err := migrateInbound(inboundConfig)
		if err != nil {
			tag := inboundConfig.Tag
			if tag == "" {
				tag = format.ToString(i)
			}
			logger.Warn("ignoring inbound ", tag, ": ", err)
			continue
		}
		options.Inbounds = append(options.Inbounds, inbound)
	}
	outboundServerRule := option.DNSRule{
		Type: C.RuleTypeDefault,
		DefaultOptions: option.DefaultDNSRule{
			Server: "local",
		},
	}
	for i, outboundConfig := range v2rayConfig.OutboundConfigs {
		outbound, err := migrateOutbound(outboundConfig, &outboundServerRule.DefaultOptions)
		if err != nil {
			tag := outboundConfig.Tag
			if tag == "" {
				tag = format.ToString(i)
			}
			logger.Warn("ignoring outbound ", tag, ": ", err)
			continue
		}
		options.Outbounds = append(options.Outbounds, outbound)
	}
	if len(outboundServerRule.DefaultOptions.Domain) > 0 {
		options.DNS.Rules = append(options.DNS.Rules, outboundServerRule)
	}
	if routerConfig := v2rayConfig.RouterConfig; routerConfig != nil {
		for _, ruleMessage := range routerConfig.RuleList {
			rule, err := migrateRule(ruleMessage)
			if err != nil {
				logger.Warn("ignoring rule: ", err)
				continue
			}
			if options.Route == nil {
				options.Route = &option.RouteOptions{}
			}
			options.Route.Rules = append(options.Route.Rules, rule)
		}
	}
	return options, nil
}
