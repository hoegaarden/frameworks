package integration

import (
	"fmt"
	"net/url"
)

// ControlPlane is a struct that knows how to start your test control plane.
//
// A minimal control plane consists of an Etcd and an APIServer.
//
// Additionall control plane components can be added as `AdditionalComponents`.
// These will be started and stopped with the control plane when it is brought
// up and teared down.
type ControlPlane struct {
	APIServer            *APIServer
	Etcd                 *Etcd
	AdditionalComponents []ControlPlaneComponent
}

// ControlPlaneComponent is an additional component that can be added to the
// control plane.
//
// It is the responsibility of a component to configure itself to be part
// of the control plane. This can be done in `RegisterTo()` method. This method
// is passing the control plane in question, so the component has access to the
// control it should connect to and can query it for the configuration needed.
type ControlPlaneComponent interface {
	Start(RemoteConnectionConfig) error
	Stop() error
	ConnectionConfig() (RemoteConnectionConfig, error)
}

// Start will start your control plane processes. To stop them, call Stop().
func (f *ControlPlane) Start() error {
	var err error

	if f.Etcd == nil {
		f.Etcd = &Etcd{}
	}
	if err = f.Etcd.Start(); err != nil {
		return err
	}
	etcdConnectionConf, err := f.Etcd.ConnectionConfig()
	if err != nil {
		return err
	}

	if f.APIServer == nil {
		f.APIServer = &APIServer{}
	}
	if err = f.APIServer.Start(etcdConnectionConf); err != nil {
		return err
	}
	apiServerConnectionConf, err := f.APIServer.ConnectionConfig()
	if err != nil {
		return err
	}

	for _, c := range f.AdditionalComponents {
		if err := c.Start(apiServerConnectionConf); err != nil {
			return err
		}
	}

	return nil
}

// Stop will stop your control plane processes, and clean up their data.
func (f *ControlPlane) Stop() error {
	for _, c := range f.AdditionalComponents {
		if err := c.Stop(); err != nil {
			return nil
		}
	}

	if f.APIServer != nil {
		if err := f.APIServer.Stop(); err != nil {
			return err
		}
	}
	if f.Etcd != nil {
		if err := f.Etcd.Stop(); err != nil {
			return err
		}
	}
	return nil
}

// APIURL returns the URL you should connect to to talk to your API.
func (f *ControlPlane) APIURL() *url.URL {
	apiConnectionConfig, err := f.APIServer.ConnectionConfig()
	if err != nil {
		// TODO handle empty URL properly, catch error
		return &url.URL{}
	}

	return apiConnectionConfig.URL
}

// KubeCtl returns a pre-configured KubeCtl, ready to connect to this
// ControlPlane.
func (f *ControlPlane) KubeCtl() *KubeCtl {
	k := &KubeCtl{}
	k.Opts = append(k.Opts, fmt.Sprintf("--server=%s", f.APIURL()))
	return k
}
