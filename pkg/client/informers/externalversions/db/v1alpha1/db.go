/*
Copyright 2017 The Kubernetes Authors.

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

// This file was automatically generated by informer-gen

package v1alpha1

import (
	time "time"

	db_v1alpha1 "github.com/MYOB-Technology/ops-kube-db-operator/pkg/apis/db/v1alpha1"
	versioned "github.com/MYOB-Technology/ops-kube-db-operator/pkg/client/clientset/versioned"
	internalinterfaces "github.com/MYOB-Technology/ops-kube-db-operator/pkg/client/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/MYOB-Technology/ops-kube-db-operator/pkg/client/listers/db/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// DBInformer provides access to a shared informer and lister for
// DBs.
type DBInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.DBLister
}

type dBInformer struct {
	factory internalinterfaces.SharedInformerFactory
}

// NewDBInformer constructs a new informer for DB type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewDBInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				return client.DbV1alpha1().DBs(namespace).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				return client.DbV1alpha1().DBs(namespace).Watch(options)
			},
		},
		&db_v1alpha1.DB{},
		resyncPeriod,
		indexers,
	)
}

func defaultDBInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewDBInformer(client, v1.NamespaceAll, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
}

func (f *dBInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&db_v1alpha1.DB{}, defaultDBInformer)
}

func (f *dBInformer) Lister() v1alpha1.DBLister {
	return v1alpha1.NewDBLister(f.Informer().GetIndexer())
}
