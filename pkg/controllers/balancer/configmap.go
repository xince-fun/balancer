package controller

import (
	"context"
	"fmt"
	exposerv1 "github.com/xince-fun/balancer/pkg/apis/balancer/v1"
	"github.com/xince-fun/balancer/pkg/controllers/balancer/nginx"
	"hash"
	"hash/fnv"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/dump"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func NewConfigMap(balancer *exposerv1.Balancer) (*corev1.ConfigMap, error) {
	return &corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      ConfigMapName(balancer),
			Namespace: balancer.Namespace,
		},
		Data: map[string]string{
			"nginx.conf": nginx.NewConfig(balancer),
		},
	}, nil
}

func (r *BalancerReconciler) syncConfigMap(balancer *exposerv1.Balancer) (*corev1.ConfigMap, error) {
	cm, err := NewConfigMap(balancer)
	if err != nil {
		return nil, err
	}

	// set controller as the controller owner-reference of cm
	if err := controllerutil.SetControllerReference(balancer, cm, r.Scheme); err != nil {
		return nil, err
	}

	foundCm := &corev1.ConfigMap{}
	err = r.Get(context.Background(), types.NamespacedName{Namespace: cm.Namespace, Name: cm.Name}, foundCm)
	if err != nil && errors.IsNotFound(err) {
		// corresponding cm not foundCm in the cluster, create it with the newest cm
		if err = r.Create(context.Background(), cm); err != nil {
			return nil, err
		}
		log.Info("Sync ConfigMap", cm.Name, "created")
		return cm, nil
	} else if err != nil {
		return nil, err
	}

	// corresponding com foundCm, update it with the newest cm
	foundCm.Data = cm.Data
	if err = r.Update(context.Background(), foundCm); err != nil {
		return nil, err
	}
	log.Info("Sync ConfigMap", foundCm.Name, "updated")
	return cm, nil
}

func ConfigMapName(balancer *exposerv1.Balancer) string {
	return balancer.Name + "-proxy-configmap"
}

func ConfigMapHash(cm *corev1.ConfigMap) string {
	hasher := fnv.New32a()
	DeepHashObject(hasher, cm)
	return rand.SafeEncodeString(fmt.Sprint(hasher.Sum32()))
}

// DeepHashObject writes specified object to hash using the spew library
// which follows pointers and prints actual values of the nested objects
// ensuring the hash does not change when a pointer changes.
func DeepHashObject(hasher hash.Hash, objectToWrite interface{}) {
	hasher.Reset()
	fmt.Fprintf(hasher, "%v", dump.ForHash(objectToWrite))
}
