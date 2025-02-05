package options

import (
	"net"

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/discovery/aggregated"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
)

const defaultEtcdPathPrefix = "/registry/grafana.app"

type Options struct {
	RecommendedOptions *genericoptions.RecommendedOptions
	AggregatorOptions  *AggregatorServerOptions
	StorageOptions     *StorageOptions
	ExtraOptions       *ExtraOptions
}

func NewOptions(codec runtime.Codec) *Options {
	return &Options{
		RecommendedOptions: genericoptions.NewRecommendedOptions(
			defaultEtcdPathPrefix,
			codec,
		),
		AggregatorOptions: NewAggregatorServerOptions(),
		StorageOptions:    NewStorageOptions(),
		ExtraOptions:      NewExtraOptions(),
	}
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	o.RecommendedOptions.AddFlags(fs)
	o.AggregatorOptions.AddFlags(fs)
	o.StorageOptions.AddFlags(fs)
	o.ExtraOptions.AddFlags(fs)
}

func (o *Options) Validate() []error {
	if errs := o.ExtraOptions.Validate(); len(errs) != 0 {
		return errs
	}

	if errs := o.StorageOptions.Validate(); len(errs) != 0 {
		return errs
	}

	if errs := o.AggregatorOptions.Validate(); len(errs) != 0 {
		return errs
	}

	if errs := o.RecommendedOptions.SecureServing.Validate(); len(errs) != 0 {
		return errs
	}

	if o.ExtraOptions.DevMode {
		// NOTE: Only consider authn for dev mode - resolves the failure due to missing extension apiserver auth-config
		// in parent k8s
		if errs := o.RecommendedOptions.Authentication.Validate(); len(errs) != 0 {
			return errs
		}
	}

	if o.StorageOptions.StorageType == StorageTypeEtcd {
		if errs := o.RecommendedOptions.Etcd.Validate(); len(errs) != 0 {
			return errs
		}
	}

	return nil
}

func (o *Options) ApplyTo(serverConfig *genericapiserver.RecommendedConfig) error {
	serverConfig.AggregatedDiscoveryGroupManager = aggregated.NewResourceManager("apis")

	if err := o.ExtraOptions.ApplyTo(serverConfig); err != nil {
		return err
	}

	if !o.ExtraOptions.DevMode {
		o.RecommendedOptions.SecureServing.Listener = newFakeListener()
	}

	if err := o.RecommendedOptions.SecureServing.ApplyTo(&serverConfig.SecureServing, &serverConfig.LoopbackClientConfig); err != nil {
		return err
	}

	if o.ExtraOptions.DevMode {
		// NOTE: Only consider authn for dev mode - resolves the failure due to missing extension apiserver auth-config
		// in parent k8s
		if err := o.RecommendedOptions.Authentication.ApplyTo(&serverConfig.Authentication, serverConfig.SecureServing, serverConfig.OpenAPIConfig); err != nil {
			return err
		}
	}

	if !o.ExtraOptions.DevMode {
		if err := serverConfig.SecureServing.Listener.Close(); err != nil {
			return err
		}
		serverConfig.SecureServing = nil
	}

	return nil
}

type fakeListener struct {
	server net.Conn
	client net.Conn
}

func newFakeListener() *fakeListener {
	server, client := net.Pipe()
	return &fakeListener{
		server: server,
		client: client,
	}
}

func (f *fakeListener) Accept() (net.Conn, error) {
	return f.server, nil
}

func (f *fakeListener) Close() error {
	if err := f.client.Close(); err != nil {
		return err
	}
	return f.server.Close()
}

func (f *fakeListener) Addr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 3000, Zone: ""}
}
