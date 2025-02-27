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
// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/bpfd-dev/bpfd/bpfd-operator/apis/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// TcProgramLister helps list TcPrograms.
// All objects returned here must be treated as read-only.
type TcProgramLister interface {
	// List lists all TcPrograms in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.TcProgram, err error)
	// Get retrieves the TcProgram from the index for a given name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.TcProgram, error)
	TcProgramListerExpansion
}

// tcProgramLister implements the TcProgramLister interface.
type tcProgramLister struct {
	indexer cache.Indexer
}

// NewTcProgramLister returns a new TcProgramLister.
func NewTcProgramLister(indexer cache.Indexer) TcProgramLister {
	return &tcProgramLister{indexer: indexer}
}

// List lists all TcPrograms in the indexer.
func (s *tcProgramLister) List(selector labels.Selector) (ret []*v1alpha1.TcProgram, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.TcProgram))
	})
	return ret, err
}

// Get retrieves the TcProgram from the index for a given name.
func (s *tcProgramLister) Get(name string) (*v1alpha1.TcProgram, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("tcprogram"), name)
	}
	return obj.(*v1alpha1.TcProgram), nil
}
