package controller

import (
	"context"
	exposerv1 "github.com/xince-fun/balancer/pkg/apis/balancer/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *BalancerReconciler) syncFrontendService(balancer *exposerv1.Balancer) error {
	svc, err := NewFrontendService(balancer)
	if err != nil {
		return err
	}

	// set controller as the controller owner-reference of svc
	if err := controllerutil.SetControllerReference(balancer, svc, r.Scheme); err != nil {
		return err
	}

	foundSvc := &corev1.Service{}
	err = r.Get(context.Background(), types.NamespacedName{Namespace: svc.Namespace, Name: svc.Name}, foundSvc)
	if err != nil && errors.IsNotFound(err) {
		// corresponding service not found in the cluster, create it with the newest svc
		if err = r.Create(context.Background(), svc); err != nil {
			return err
		}
		log.Info("Sync Frontend Service", svc.Name, "created")
		return nil
	} else if err != nil {
		return err
	}

	foundSvc.Spec.Ports = svc.Spec.Ports
	foundSvc.Spec.Selector = svc.Spec.Selector
	if err = r.Update(context.Background(), foundSvc); err != nil {
		return err
	}
	log.Info("Sync Frontend Service", foundSvc.Name, "updated")
	return nil
}

// NewFrontendService creates a new front-end Service for handling all requests incoming.
// All the incoming requests will be forwarded to backend services by the nginx instance.
func NewFrontendService(balancer *exposerv1.Balancer) (*corev1.Service, error) {
	var balancerPorts []corev1.ServicePort
	for _, port := range balancer.Spec.Ports {
		balancerPorts = append(balancerPorts, corev1.ServicePort{
			Name:     port.Name,
			Protocol: corev1.Protocol(port.Protocol),
			Port:     int32(port.Port),
		})
	}
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      balancer.Name,
			Namespace: balancer.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: NewPodLabels(balancer),
			Type:     corev1.ServiceTypeClusterIP,
			Ports:    balancerPorts,
		},
	}, nil
}
