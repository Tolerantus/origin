package legacyclient

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	kapi "k8s.io/kubernetes/pkg/api"
	clientscheme "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/scheme"

	oclient "github.com/openshift/origin/pkg/client"
	securityapi "github.com/openshift/origin/pkg/security/api"
	securityapiinstall "github.com/openshift/origin/pkg/security/api/install"
)

// if this is being used, we need to be sure that the core API client has our types in the scheme
func init() {
	securityapiinstall.InstallIntoDeprecatedV1(clientscheme.GroupFactoryRegistry, clientscheme.Registry, clientscheme.Scheme)
}

// New creates a legacy client for SCC access.  This only exists for `oc` compatibility with old servers
func New(c *rest.Config) (SecurityContextConstraintInterface, error) {
	config := *c
	if err := oclient.SetOpenShiftDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}

	return &securityContextConstraint{client}, nil
}

// New creates a legacy client for SCC access.  This only exists for `oc` compatibility with old servers
func NewFromClient(client rest.Interface) SecurityContextConstraintInterface {
	return &securityContextConstraint{client}
}

// SecurityContextConstraintInterface exposes methods on SecurityContextConstraints resources
type SecurityContextConstraintInterface interface {
	List(opts metav1.ListOptions) (*securityapi.SecurityContextConstraintsList, error)
	Get(name string, options metav1.GetOptions) (*securityapi.SecurityContextConstraints, error)
	Create(*securityapi.SecurityContextConstraints) (*securityapi.SecurityContextConstraints, error)
	Update(*securityapi.SecurityContextConstraints) (*securityapi.SecurityContextConstraints, error)
	Delete(name string) error
	Watch(opts metav1.ListOptions) (watch.Interface, error)
}

type securityContextConstraint struct {
	r rest.Interface
}

func (c *securityContextConstraint) List(opts metav1.ListOptions) (result *securityapi.SecurityContextConstraintsList, err error) {
	result = &securityapi.SecurityContextConstraintsList{}
	err = c.r.Get().Resource("securitycontextconstraints").VersionedParams(&opts, kapi.ParameterCodec).Do().Into(result)
	return
}

func (c *securityContextConstraint) Get(name string, options metav1.GetOptions) (result *securityapi.SecurityContextConstraints, err error) {
	result = &securityapi.SecurityContextConstraints{}
	err = c.r.Get().Resource("securitycontextconstraints").Name(name).VersionedParams(&options, kapi.ParameterCodec).Do().Into(result)
	return
}

func (c *securityContextConstraint) Create(scc *securityapi.SecurityContextConstraints) (result *securityapi.SecurityContextConstraints, err error) {
	result = &securityapi.SecurityContextConstraints{}
	err = c.r.Post().Resource("securitycontextconstraints").Body(scc).Do().Into(result)
	return
}

func (c *securityContextConstraint) Update(scc *securityapi.SecurityContextConstraints) (result *securityapi.SecurityContextConstraints, err error) {
	result = &securityapi.SecurityContextConstraints{}
	err = c.r.Put().Resource("securitycontextconstraints").Name(scc.Name).Body(scc).Do().Into(result)
	return
}

func (c *securityContextConstraint) Delete(name string) (err error) {
	err = c.r.Delete().Resource("securitycontextconstraints").Name(name).Do().Error()
	return
}

func (c *securityContextConstraint) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return c.r.Get().Prefix("watch").Resource("securitycontextconstraints").VersionedParams(&opts, kapi.ParameterCodec).Watch()
}
