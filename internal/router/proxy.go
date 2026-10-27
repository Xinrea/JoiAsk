package router

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// Match the local/container proxy networks trusted by the bundled Caddy setup,
// rather than Gin's default of trusting every peer and every forwarded IP.
// Deployments with a different boundary can set security.trusted_proxies to
// explicit proxy IPs/CIDRs, or [] to ignore forwarded headers entirely.
func configureTrustedProxies(r *gin.Engine) error {
	proxies := []string{
		"127.0.0.0/8", "::1/128",
		"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16",
		"fc00::/7", "fe80::/10",
	}
	if viper.IsSet("security.trusted_proxies") {
		proxies = viper.GetStringSlice("security.trusted_proxies")
	}
	return r.SetTrustedProxies(proxies)
}
