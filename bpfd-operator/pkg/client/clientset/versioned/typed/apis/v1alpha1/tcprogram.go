/*
Copyright 2023 The bpfd Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "github.com/bpfd-dev/bpfd/bpfd-operator/apis/v1alpha1"
	scheme "github.com/bpfd-dev/bpfd/bpfd-operator/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// TcProgramsGetter has a method to return a TcProgramInterface.
// A group's client should implement this interface.
type TcProgramsGetter interface {
	TcPrograms() TcProgramInterface
}

// TcProgramInterface has methods to work with TcProgram resources.
type TcProgramInterface interface {
	Create(ctx context.Context, tcProgram *v1alpha1.TcProgram, opts v1.CreateOptions) (*v1alpha1.TcProgram, error)
	Update(ctx context.Context, tcProgram *v1alpha1.TcProgram, opts v1.UpdateOptions) (*v1alpha1.TcProgram, error)
	UpdateStatus(ctx context.Context, tcProgram *v1alpha1.TcProgram, opts v1.UpdateOptions) (*v1alpha1.TcProgram, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.TcProgram, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.TcProgramList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.TcProgram, err error)
	TcProgramExpansion
}

// tcPrograms implements TcProgramInterface
type tcPrograms struct {
	client rest.Interface
}

// newTcPrograms returns a TcPrograms
func newTcPrograms(c *BpfdV1alpha1Client) *tcPrograms {
	return &tcPrograms{
		client: c.RESTClient(),
	}
}

// Get takes name of the tcProgram, and returns the corresponding tcProgram object, and an error if there is any.
func (c *tcPrograms) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.TcProgram, err error) {
	result = &v1alpha1.TcProgram{}
	err = c.client.Get().
		Resource("tcprograms").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of TcPrograms that match those selectors.
func (c *tcPrograms) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.TcProgramList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.TcProgramList{}
	err = c.client.Get().
		Resource("tcprograms").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested tcPrograms.
func (c *tcPrograms) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("tcprograms").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a tcProgram and creates it.  Returns the server's representation of the tcProgram, and an error, if there is any.
func (c *tcPrograms) Create(ctx context.Context, tcProgram *v1alpha1.TcProgram, opts v1.CreateOptions) (result *v1alpha1.TcProgram, err error) {
	result = &v1alpha1.TcProgram{}
	err = c.client.Post().
		Resource("tcprograms").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(tcProgram).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a tcProgram and updates it. Returns the server's representation of the tcProgram, and an error, if there is any.
func (c *tcPrograms) Update(ctx context.Context, tcProgram *v1alpha1.TcProgram, opts v1.UpdateOptions) (result *v1alpha1.TcProgram, err error) {
	result = &v1alpha1.TcProgram{}
	err = c.client.Put().
		Resource("tcprograms").
		Name(tcProgram.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(tcProgram).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *tcPrograms) UpdateStatus(ctx context.Context, tcProgram *v1alpha1.TcProgram, opts v1.UpdateOptions) (result *v1alpha1.TcProgram, err error) {
	result = &v1alpha1.TcProgram{}
	err = c.client.Put().
		Resource("tcprograms").
		Name(tcProgram.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(tcProgram).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the tcProgram and deletes it. Returns an error if one occurs.
func (c *tcPrograms) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("tcprograms").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *tcPrograms) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("tcprograms").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched tcProgram.
func (c *tcPrograms) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.TcProgram, err error) {
	result = &v1alpha1.TcProgram{}
	err = c.client.Patch(pt).
		Resource("tcprograms").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
