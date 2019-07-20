package appregistry

import (
	"net/url"

	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	apprclient "github.com/operator-framework/go-appr/appregistry"
	"github.com/operator-framework/operator-marketplace/pkg/proxy"
	"net/http"
)

// NewClientFactory return a factory which can be used to instantiate a new appregistry client
func NewClientFactory() ClientFactory {
	return &factory{}
}

type Options struct {
	// Source refers to the URL of the remote app registry server.
	Source string

	// AuthToken refers to the authorization token required to access operator
	// manifest in private repositories.
	//
	// If not set, it is assumed that the remote registry is public.
	AuthToken string
}

// ClientFactory is an interface that wraps the New method.
//
// New returns a new instance of appregistry Client from the specified source.
type ClientFactory interface {
	New(options Options) (Client, error)
}

type factory struct{}

func (f *factory) New(options Options) (Client, error) {
	u, err := url.Parse(options.Source)
	if err != nil {
		return nil, err
	}

	// A custom HTTPClient is used here since the default HTTPClients ProxyFromEnvironment
	// uses a cache which won't let us update the proxy env vars.
	cli := &http.Client{
		Transport: &http.Transport{
			Proxy: func(req *http.Request) (*url.URL, error) {
				return proxy.GetInstance().GetProxyConfig().ProxyFunc()(req.URL)
			},
		},
	}

	transport := httptransport.NewWithClient(u.Host, u.Path, []string{u.Scheme}, cli)
	transport.Consumers["application/x-gzip"] = runtime.ByteStreamConsumer()

	// If a bearer token has been specified then we should pass it along in the headers
	if options.AuthToken != "" {
		tokenAuthWriter := httptransport.APIKeyAuth("Authorization", "header", options.AuthToken)
		transport.DefaultAuthentication = tokenAuthWriter
	}

	c := apprclient.New(transport, strfmt.Default)

	return &client{
		adapter: &apprApiAdapterImpl{client: c},
		decoder: &blobDecoderImpl{},
	}, nil
}
