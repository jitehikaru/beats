// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package metadata

import (
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
	"github.com/elastic/beats/v7/libbeat/common/safemapstr"
)

type service struct {
	store    cache.Store
	resource *Resource
}

// NewServiceMetadataGenerator creates a metagen for service resources
func NewServiceMetadataGenerator(cfg *common.Config, services cache.Store, namespace MetaGen, client k8s.Interface) MetaGen {
	return &service{
		resource: NewNamespaceAwareResourceMetadataGenerator(cfg, client, namespace),
		store:    services,
	}
}

// Generate generates service metadata from a resource object
// Metadata map is in the following form:
//
//	{
//		  "kubernetes": {},
//	   "some.ecs.field": "asdf"
//	}
//
// All Kubernetes fields that need to be stored under kuberentes. prefix are populetad by
// GenerateK8s method while fields that are part of ECS are generated by GenerateECS method
func (s *service) Generate(obj kubernetes.Resource, opts ...FieldOptions) common.MapStr {
	ecsFields := s.GenerateECS(obj)
	meta := common.MapStr{
		"kubernetes": s.GenerateK8s(obj, opts...),
	}
	meta.DeepUpdate(ecsFields)
	return meta
}

// GenerateECS generates service ECS metadata from a resource object
func (s *service) GenerateECS(obj kubernetes.Resource) common.MapStr {
	return s.resource.GenerateECS(obj)
}

// GenerateK8s generates service metadata from a resource object
func (s *service) GenerateK8s(obj kubernetes.Resource, opts ...FieldOptions) common.MapStr {
	svc, ok := obj.(*kubernetes.Service)
	if !ok {
		return nil
	}

	out := s.resource.GenerateK8s("service", obj, opts...)

	selectors := svc.Spec.Selector
	if len(selectors) == 0 {
		return out
	}
	svcMap := GenerateMap(selectors, s.resource.config.LabelsDedot)
	if len(svcMap) != 0 {
		safemapstr.Put(out, "selectors", svcMap)
	}

	return out
}

// GenerateFromName generates pod metadata from a service name
func (s *service) GenerateFromName(name string, opts ...FieldOptions) common.MapStr {
	if s.store == nil {
		return nil
	}

	if obj, ok, _ := s.store.GetByKey(name); ok {
		svc, ok := obj.(*kubernetes.Service)
		if !ok {
			return nil
		}

		return s.GenerateK8s(svc, opts...)
	}

	return nil
}
