package mimirtool

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/grafana/dskit/crypto/tls"
	mimirtool "github.com/grafana/mimir/pkg/mimirtool/client"
)

var (
	storeRulesSHA256 bool
)

func init() {
	// Set descriptions to support markdown syntax, this will be used in document generation
	// and the language server.
	schema.DescriptionKind = schema.StringMarkdown

	// Customize the content of descriptions when output. For example you can add defaults on
	// to the exported descriptions if present.
	// schema.SchemaDescriptionBuilder = func(s *schema.Schema) string {
	// 	desc := s.Description
	// 	if s.Default != nil {
	// 		desc += fmt.Sprintf(" Defaults to `%v`.", s.Default)
	// 	}
	// 	return strings.TrimSpace(desc)
	// }
}

// New returns a newly created provider
func New(version string, mimirClient mimirClientInterface) func() *schema.Provider {
	return func() *schema.Provider {
		p := &schema.Provider{
			Schema: map[string]*schema.Schema{
				"url": {
					Type:         schema.TypeString,
					Required:     true,
					DefaultFunc:  schema.EnvDefaultFunc("MIMIR_ADDRESS", nil),
					Description:  "Address to use when contacting Grafana Mimir. May alternatively be set via the `MIMIR_ADDRESS` environment variable.",
					ValidateFunc: validation.IsURLWithHTTPorHTTPS,
				},
				"tenant_id": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("MIMIR_TENANT_ID", nil),
					Description: "Tenant ID to use when contacting Grafana Mimir. May alternatively be set via the `MIMIR_TENANT_ID` environment variable.",
				},
				"user": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("MIMIR_API_USER.", nil),
					Description: "API user to use when contacting Grafana Mimir. May alternatively be set via the `MIMIR_API_USER` environment variable.",
				},
				"key": {
					Type:        schema.TypeString,
					Optional:    true,
					Sensitive:   true,
					DefaultFunc: schema.EnvDefaultFunc("MIMIR_API_KEY", ""),
					Description: "API key to use when contacting Grafana Mimir. May alternatively be set via the `MIMIR_API_KEY` environment variable.",
				},
				"token": {
					Type:        schema.TypeString,
					Optional:    true,
					Sensitive:   true,
					DefaultFunc: schema.EnvDefaultFunc("MIMIR_AUTH_TOKEN.", nil),
					Description: "Authentication token for bearer token or JWT auth when contacting Grafana Mimir. May alternatively be set via the `MIMIR_AUTH_TOKEN` environment variable.",
				},
				"tls_key_path": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("MIMIR_TLS_KEY_PATH", nil),
					Description: "Client TLS key file to use to authenticate to the MIMIR server. May alternatively be set via the `MIMIR_TLS_KEY_PATH` environment variable.",
				},
				"tls_cert_path": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("MIMIR_TLS_CERT_PATH", nil),
					Description: "Client TLS certificate file to use to authenticate to the MIMIR server. May alternatively be set via the `MIMIR_TLS_CERT_PATH` environment variable.",
				},
				"ca_cert_path": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("MIMIR_CA_CERT_PATH", nil),
					Description: "Certificate CA bundle to use to verify the MIMIR server's certificate. May alternatively be set via the `MIMIR_CA_CERT_PATH` environment variable.",
				},
				"insecure_skip_verify": {
					Type:        schema.TypeBool,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("MIMIR_INSECURE_SKIP_VERIFY", nil),
					Description: "Skip TLS certificate verification. May alternatively be set via the `MIMIR_INSECURE_SKIP_VERIFY` environment variable.",
				},
				"prometheus_http_prefix": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("MIMIR_API_PREFIX", "/prometheus"),
					Description: "Path prefix to use for rules. May alternatively be set via the `MIMIR_API_PREFIX` environment variable.",
				},
				"alertmanager_http_prefix": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("MIMIR_ALERTMANAGER_HTTP_PREFIX", "/alertmanager"),
					Description: "Path prefix to use for alertmanager. May alternatively be set via the `MIMIR_ALERTMANAGER_HTTP_PREFIX` environment variable.",
				},
				"store_rules_sha256": {
					Type:        schema.TypeBool,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("MIMIR_STORE_RULES_SHA256", false),
					Description: "Set to true if you want to save only the sha256sum instead of namespace's groups rules definition in the tfstate.",
				},
			},
			DataSourcesMap: map[string]*schema.Resource{},
			ResourcesMap: map[string]*schema.Resource{
				"mimirtool_ruler_namespace": resourceRulerNamespace(),
				"mimirtool_alertmanager":    resourceAlertManager(),
			},
		}

		p.ConfigureContextFunc = configure(version, p, mimirClient)

		return p
	}
}

func configure(version string, p *schema.Provider, mimirClient mimirClientInterface) func(context.Context, *schema.ResourceData) (interface{}, diag.Diagnostics) {
	return func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		var (
			diags diag.Diagnostics
			err   error
		)
		p.UserAgent("terraform-provider-mimirtool", version)

		c := &client{}
		if mimirClient != nil {
			c.cli = mimirClient
		} else {
			c.cli, err = getDefaultMimirClient(d)
			if err != nil {
				return nil, diag.FromErr(err)
			}
		}

		storeRulesSHA256 = d.Get("store_rules_sha256").(bool)
		return c, diags
	}
}

func getDefaultMimirClient(d *schema.ResourceData) (mimirClientInterface, error) {
	return mimirtool.New(mimirtool.Config{
		AuthToken: d.Get("token").(string),
		User:      d.Get("user").(string),
		Key:       d.Get("key").(string),
		Address:   d.Get("url").(string),
		ID:        d.Get("tenant_id").(string),
		TLS: tls.ClientConfig{
			CAPath:             d.Get("ca_cert_path").(string),
			CertPath:           d.Get("tls_cert_path").(string),
			KeyPath:            d.Get("tls_key_path").(string),
			InsecureSkipVerify: d.Get("insecure_skip_verify").(bool),
		},
	})
}
