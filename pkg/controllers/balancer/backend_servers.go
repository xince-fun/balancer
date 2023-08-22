package controller

import (
	"context"
	"fmt"
	exposerv1 "github.com/xince-fun/balancer/pkg/apis/balancer/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sync"
)

func (r *BalancerReconciler) syncBackendServices(balancer *exposerv1.Balancer) error {
	// get current backend services
	var svcList corev1.ServiceList
	if err := r.List(context.Background(), &svcList, client.MatchingLabels(NewServiceLabels(balancer))); err != nil {
		return err
	}

	backendServicesToCreate, backendServicesToDelete, _ := groupBackendServers(balancer, svcList.Items)

	wg := sync.WaitGroup{}

	// start coroutines to delete services-to-be-deleted
	deleteErrCh := make(chan error, len(backendServicesToDelete))
	wg.Add(len(backendServicesToDelete))
	// This will cause error!
	// for _, svcToDelete := range backendServicesToDelete {
	//     ...
	// }
	for i := range backendServicesToDelete {
		svcToDelete := backendServicesToDelete[i]
		go func(svc *corev1.Service) {
			defer wg.Done()
			if err := r.Delete(context.Background(), svc); err != nil {
				deleteErrCh <- err
			}
		}(&svcToDelete)
	}
	wg.Wait()

	// start coroutines to create services-to-be-created
	createErrCh := make(chan error, len(backendServicesToCreate))
	wg.Add(len(backendServicesToCreate))

	for i := range backendServicesToCreate {
		svcToCreate := backendServicesToCreate[i]
		go func(svc *corev1.Service) {
			defer wg.Done()

			// set controller as the controller owner-reference of svc
			if err := controllerutil.SetControllerReference(balancer, svc, r.Scheme); err != nil {
				createErrCh <- err
				return
			}

			// create or update
			foundSvc := &corev1.Service{}
			err := r.Get(context.Background(), types.NamespacedName{Namespace: svc.Namespace, Name: svc.Name}, foundSvc)
			if err != nil && errors.IsNotFound(err) {
				err = r.Create(context.Background(), svc)
				if err != nil {
					createErrCh <- err
				} else {
					log.Info("Sync Backend Services", svc.Name, "created")
				}
				return
			} else if err != nil {
				createErrCh <- err
				return
			}

			foundSvc.Spec.Ports = svc.Spec.Ports
			foundSvc.Spec.Selector = svc.Spec.Selector
			err = r.Update(context.Background(), foundSvc)
			if err != nil {
				createErrCh <- err
			} else {
				log.Info("Sync Backend Services", foundSvc.Name, "updated")
			}
			return
		}(&svcToCreate)
	}
	wg.Wait()

	select {
	case err := <-deleteErrCh:
		return err
	case err := <-createErrCh:
		return err
	default:
		return r.syncBalancerStatus(balancer)

	}
}

func (r *BalancerReconciler) syncBalancerStatus(balancer *exposerv1.Balancer) error {
	// get current backend services
	var svcList corev1.ServiceList
	if err := r.List(context.Background(), &svcList, client.MatchingFields(NewServiceLabels(balancer))); err != nil {
		return err
	}

	_, backendServicesToDelete, activeBackendServices := groupBackendServers(balancer, svcList.Items)

	actualStatus := exposerv1.BalancerStatus{
		ActiveBackendsNum:   int32(len(activeBackendServices)),
		ObsoleteBackendsNum: int32(len(backendServicesToDelete)),
	}
	// nothing to do, return directly
	if reflect.DeepEqual(balancer.Status, actualStatus) {
		return nil
	}

	// status updating is required (note the assignment direction is opposite!)
	newBalancer := balancer
	newBalancer.Status = actualStatus
	return r.Status().Update(context.Background(), newBalancer)
}

// groupBackendServers gets to-be-created backend services, to-be-deleted backend services,
// and backend services which should keep unchanged according to controller and currentBackendServices in cluster.
func groupBackendServers(balancer *exposerv1.Balancer, currentBackendServices []corev1.Service) (
	backendServicesToCreate, backendServicesToDelete, activeBackendServices []corev1.Service) {
	var balancerPorts []corev1.ServicePort
	for _, port := range balancer.Spec.Ports {
		balancerPorts = append(balancerPorts, corev1.ServicePort{
			Name:       port.Name,
			Protocol:   corev1.Protocol(port.Protocol),
			Port:       int32(port.Port), // exposed port of each backend service
			TargetPort: port.TargetPort,  // exposed port of the outside pods
		})
	}

	// create each backend service
	for _, backend := range balancer.Spec.Backends {
		// selector example: {app: test, version: v1}
		// which is used to select one specifiec outside Pod
		selector := map[string]string{}
		for k, v := range balancer.Spec.Selector {
			selector[k] = v
		}
		for k, v := range backend.Selector {
			selector[k] = v
		}
		backendServicesToCreate = append(backendServicesToCreate, corev1.Service{
			ObjectMeta: v1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s-backend", balancer.Name, backend.Name),
				Namespace: balancer.Namespace,
				Labels:    NewServiceLabels(balancer), // for annotating this is a service belongs to a controller
			},
			Spec: corev1.ServiceSpec{
				Selector: selector,
				Type:     corev1.ServiceTypeClusterIP,
				Ports:    balancerPorts,
			},
		})
	}

	for _, svc := range currentBackendServices {
		// svc is a currently running service in cluster.
		// We need to judge whether svc should be active according to the expected controller object.
		// If yes, we just add it to activeBackendServices (which means it should keep running without change);
		// Otherwise, we add it to backendServiceToDelete (whihc means it will be deleted soon).
		existActiveSvc := false
		for _, svcToCreate := range backendServicesToCreate {
			if svc.Name == svcToCreate.Name && svc.Namespace == svcToCreate.Namespace {
				activeBackendServices = append(activeBackendServices, svc)
				existActiveSvc = true
				break
			}
		}
		if !existActiveSvc {
			backendServicesToDelete = append(backendServicesToDelete, svc)
		}
	}
	return
}
