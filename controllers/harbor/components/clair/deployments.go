package clair

import (
	"context"
	"encoding/json"
	"path"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	containerregistryv1alpha1 "github.com/ovh/harbor-operator/api/v1alpha1"
	"github.com/ovh/harbor-operator/pkg/factories/application"
	"github.com/ovh/harbor-operator/pkg/factories/logger"
)

const (
	initImage = "hairyhenderson/gomplate"
)

var (
	revisionHistoryLimit int32 = 0 // nolint:golint
	varFalse                   = false
)

func (c *Clair) GetDeployments(ctx context.Context) []*appsv1.Deployment { // nolint:funlen
	operatorName := application.GetName(ctx)
	harborName := c.harbor.GetName()

	vulnsrc, err := json.Marshal(c.harbor.Spec.Components.Clair.VulnerabilitySources)
	if err != nil {
		logger.Get(ctx).Error(err, "invalid vulnerability sources")
	}

	return []*appsv1.Deployment{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      c.harbor.NormalizeComponentName(containerregistryv1alpha1.ClairName),
				Namespace: c.harbor.Namespace,
				Labels: map[string]string{
					"app":      containerregistryv1alpha1.ClairName,
					"harbor":   harborName,
					"operator": operatorName,
				},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app":      containerregistryv1alpha1.ClairName,
						"harbor":   harborName,
						"operator": operatorName,
					},
				},
				Replicas: c.harbor.Spec.Components.Core.Replicas,
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"checksum":         c.GetConfigCheckSum(),
							"operator/version": application.GetVersion(ctx),
						},
						Labels: map[string]string{
							"app":      containerregistryv1alpha1.ClairName,
							"harbor":   harborName,
							"operator": operatorName,
						},
					},
					Spec: corev1.PodSpec{
						NodeSelector: c.harbor.Spec.Components.Clair.NodeSelector,
						Volumes: []corev1.Volume{
							{
								Name: "config-template",
								VolumeSource: corev1.VolumeSource{
									ConfigMap: &corev1.ConfigMapVolumeSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: c.harbor.NormalizeComponentName(containerregistryv1alpha1.ClairName),
										},
										Items: []corev1.KeyToPath{
											{
												Key:  configKey,
												Path: configKey,
											},
										},
									},
								},
							}, {
								Name:         "config",
								VolumeSource: corev1.VolumeSource{},
							},
						},
						InitContainers: []corev1.Container{
							{
								Name:       "configuration",
								Image:      initImage,
								WorkingDir: "/workdir",
								Args:       []string{"--input-dir", "/workdir", "--output-dir", "/processed"},
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      "config-template",
										MountPath: "/workdir",
										ReadOnly:  true,
									}, {
										Name:      "config",
										MountPath: "/processed",
										ReadOnly:  false,
									},
								},
								Env: []corev1.EnvVar{
									{
										Name:  "vulnsrc",
										Value: string(vulnsrc),
									},
								},
								EnvFrom: []corev1.EnvFromSource{
									{
										SecretRef: &corev1.SecretEnvSource{
											Optional: &varFalse,
											LocalObjectReference: corev1.LocalObjectReference{
												Name: c.harbor.Spec.Components.Clair.DatabaseSecret,
											},
										},
									},
								},
							},
						},
						Containers: []corev1.Container{
							{
								Name:  "clair",
								Image: c.harbor.Spec.Components.Clair.Image,
								Ports: []corev1.ContainerPort{
									{
										ContainerPort: 6060,
									}, {
										ContainerPort: 6061,
									},
								},

								Env: []corev1.EnvVar{
									{ // // https://github.com/goharbor/harbor/blob/master/make/photon/prepare/templates/clair/clair_env.jinja
										Name:  "HTTP_PROXY",
										Value: "",
									}, {
										Name:  "HTTPS_PROXY",
										Value: "",
									}, {
										Name:  "NO_PROXY",
										Value: "",
									}, { // https://github.com/goharbor/harbor/blob/master/make/photon/prepare/templates/clair/postgres_env.jinja
										Name: "POSTGRES_PASSWORD",
										ValueFrom: &corev1.EnvVarSource{
											SecretKeyRef: &corev1.SecretKeySelector{
												Key:      containerregistryv1alpha1.HarborClairDatabasePasswordKey,
												Optional: &varFalse,
												LocalObjectReference: corev1.LocalObjectReference{
													Name: c.harbor.Spec.Components.Clair.DatabaseSecret,
												},
											},
										},
									},
								},
								ImagePullPolicy: corev1.PullAlways,
								LivenessProbe: &corev1.Probe{
									Handler: corev1.Handler{
										HTTPGet: &corev1.HTTPGetAction{
											Path: "/health",
											Port: intstr.FromInt(6061),
										},
									},
								},
								ReadinessProbe: &corev1.Probe{
									Handler: corev1.Handler{
										HTTPGet: &corev1.HTTPGetAction{
											Path: "/health",
											Port: intstr.FromInt(6061),
										},
									},
								},
								VolumeMounts: []corev1.VolumeMount{
									{
										MountPath: path.Join("/etc/clair", configKey),
										Name:      "config",
										SubPath:   configKey,
									},
								},
							},
						},
						Priority: c.Option.Priority,
					},
				},
				RevisionHistoryLimit: &revisionHistoryLimit,
				Paused:               c.harbor.Spec.Paused,
			},
		},
	}
}
