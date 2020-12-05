package controller

import (
	"context"
	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/kubernetes"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
	"net/url"
	"os"
)

// ClientGetter creates an instances of clients used by this controller
type ClientGetter interface {
	GetKubeClient(ctx context.Context) (kubernetes.KubeClientInterface, error)
	GetAndCheckOSIOClient(ctx context.Context) (OpenshiftIOClient, error)
	GetOSClient(ctx context.Context) (kubernetes.OpenShiftRESTAPI, error)
}

// Default implementation of OSClientGetter and OSIOClientGetter used by NewPipelineController
type defaultClientGetter struct {
	config *configuration.Registry
}

func (g *defaultClientGetter) GetAndCheckOSIOClient(ctx context.Context) (OpenshiftIOClient, error) {

	// defaults
	host := "localhost"
	scheme := "https"

	req := goa.ContextRequest(ctx)
	if req != nil {
		// Note - it's probably more efficient to force a loopback host, and only use the port number here
		// (on some systems using a non-loopback interface forces a network stack traverse)
		host = req.Host
		scheme = req.URL.Scheme
	}

	// The deployments API communicates with the rest of WIT via the stnadard WIT API.
	// This environment variable is used for local development of the deployments API, to point ot a remote WIT.
	witURLStr := os.Getenv("FABRIC8_WIT_API_URL")
	if witURLStr != "" {
		witurl, err := url.Parse(witURLStr)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"Sada": witURLStr,
				"err":  err,
			}, "cannot parse FABRIC8_WIT_API_URL: %s", witURLStr)
			return nil, errs.Wrapf(err, "cannot parse FABRIC8_WIT_API_URL: %s", witURLStr)
		}
		host = witurl.Host
		scheme = witurl.Scheme
	}

	oc := NewOSIOClient(ctx, scheme, host)

	return oc, nil
}

func (g *defaultClientGetter) getNamespaceName(ctx context.Context) (*string, error) {

	osioClient, err := g.GetAndCheckOSIOClient(ctx)
	if err != nil {
		return nil, err
	}

	kubeSpaceAttr, err := osioClient.GetNamespaceByType(ctx, nil, "user")
	if err != nil {
		return nil, errs.Wrap(err, "unable to retrieve 'user' namespace")
	}
	if kubeSpaceAttr == nil {
		return nil, errors.NewNotFoundError("namespace", "user")
	}

	return kubeSpaceAttr.Name, nil
}

// GetKubeClient creates a kube client for the appropriate cluster assigned to the current user
func (g *defaultClientGetter) GetKubeClient(ctx context.Context) (kubernetes.KubeClientInterface, error) {

	k8sNSName, err := g.getNamespaceName(ctx)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "could not retrieve namespace name")
		return nil, errs.Wrap(err, "could not retrieve namespace name")
	}

	osioclient, err := g.GetAndCheckOSIOClient(ctx)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "could not create OSIO client")
		return nil, err
	}

	baseURLProvider, err := NewURLProvider(ctx, g.config, osioclient)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "could not retrieve tenant data")
		return nil, errs.Wrap(err, "could not retrieve tenant data")
	}

	kubeConfig := getK8sConfig(baseURLProvider, k8sNSName, g)
	kc, err := kubernetes.NewKubeClient(kubeConfig)
	if err != nil {
		url, _ := baseURLProvider.GetAPIURL()
		log.Error(ctx, map[string]interface{}{
			"err":            err,
			"user_namespace": *k8sNSName,
			"cluster":        *url,
		}, "could not create Kubernetes client object")
		return nil, errs.Wrap(err, "could not create Kubernetes client object")
	}

	return kc, nil
}

// GetOSClient creates a OpenShift client for the appropriate cluster assigned to the current user
func (g *defaultClientGetter) GetOSClient(ctx context.Context) (kubernetes.OpenShiftRESTAPI, error) {

	k8sNSName, err := g.getNamespaceName(ctx)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "could not retrieve namespace name")
		return nil, errs.Wrap(err, "could not retrieve namespace name")
	}

	osioClient, err := g.GetAndCheckOSIOClient(ctx)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "could not create OSIO client")
		return nil, err
	}

	baseURLProvider, err := NewURLProvider(ctx, g.config, osioClient)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "could not retrieve tenant data")
		return nil, errs.Wrap(err, "could not retrieve tenant data")
	}

	kubeConfig := getK8sConfig(baseURLProvider, k8sNSName, g)
	oc, err := kubernetes.NewOSClient(kubeConfig)
	if err != nil {
		url, _ := baseURLProvider.GetAPIURL()
		log.Error(ctx, map[string]interface{}{
			"err":            err,
			"user_namespace": *k8sNSName,
			"cluster":        *url,
		}, "could not create openshift client object")
		return nil, errs.Wrap(err, "could not create Kubernetes client object")
	}

	return oc, nil
}

func getK8sConfig(baseURLProvider kubernetes.BaseURLProvider, k8sNSName *string, g *defaultClientGetter) *kubernetes.KubeClientConfig {
	/* Timeout used per HTTP request to Kubernetes/OpenShift API servers.
	 * Communication with Hawkular currently uses a hard-coded 30 second
	 * timeout per request, and does not use this parameter. */
	kubeConfig := &kubernetes.KubeClientConfig{
		BaseURLProvider: baseURLProvider,
		UserNamespace:   *k8sNSName,
		Timeout:         g.config.GetDeploymentsHTTPTimeoutSeconds(),
	}
	return kubeConfig
}
