package controller

import (
	"context"
	exposerv1 "github.com/xince-fun/balancer/pkg/apis/balancer/v1"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *BalancerReconciler) syncDeployment(balancer *exposerv1.Balancer) error {
	// firstly, we sync configmap
	cm, err := r.syncConfigMap(balancer)
	if err != nil {
		return err
	}

	dp, err := NewDeployment(balancer)
	if err != nil {
		return err
	}
	annotations := map[string]string{
		exposerv1.ConfigMapHashKey: ConfigMapHash(cm),
	}

	// always use the newest annotations
	dp.Spec.Template.ObjectMeta.Annotations = annotations

	// set controller as the controller owner-reference of dp
	if err = controllerutil.SetControllerReference(balancer, dp, r.Scheme); err != nil {
		return err
	}

	foundDp := &appv1.Deployment{}
	err = r.Get(context.Background(), types.NamespacedName{Namespace: balancer.Namespace, Name: balancer.Name}, foundDp)
	if err != nil && errors.IsNotFound(err) {
		if err = r.Create(context.Background(), dp); err != nil {
			return err
		}
		log.Info("Sync Deployment", dp.Name, "updated")
		return nil
	} else if err != nil {
		return err
	}
	foundDp.Spec.Template = dp.Spec.Template
	if err = r.Update(context.Background(), foundDp); err != nil {
		return err
	}
	log.Info("Sync Deployment", foundDp.Name, "updated")
	return nil
}

func NewDeployment(balancer *exposerv1.Balancer) (*appv1.Deployment, error) {
	replicas := int32(1)
	labels := NewPodLabels(balancer)
	nginxContainer := corev1.Container{
		Name:  "nginx",
		Image: "nginx:latest",
		Ports: []corev1.ContainerPort{{ContainerPort: 80}},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      ConfigMapName(balancer),
				MountPath: "/etc/nginx",
				ReadOnly:  true,
			},
		},
	}
	nginxVolume := corev1.Volume{
		Name: ConfigMapName(balancer),
		VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: ConfigMapName(balancer),
			},
		}},
	}
	return &appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DeploymentName(balancer),
			Namespace: balancer.Namespace,
			Labels:    labels,
		},
		Spec: appv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      DeploymentName(balancer),
					Namespace: balancer.Namespace,
					Labels:    labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{nginxContainer},
					Volumes:    []corev1.Volume{nginxVolume},
				},
			},
		},
	}, nil
}

func DeploymentName(balancer *exposerv1.Balancer) string {
	return balancer.Name + "proxy"
}
