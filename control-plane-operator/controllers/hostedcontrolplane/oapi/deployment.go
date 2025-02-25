package oapi

import (
	"fmt"
	"github.com/openshift/hypershift/control-plane-operator/controllers/hostedcontrolplane/konnectivity"
	"net/url"
	"path"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"

	"github.com/openshift/hypershift/control-plane-operator/controllers/hostedcontrolplane/config"
	"github.com/openshift/hypershift/control-plane-operator/controllers/hostedcontrolplane/kas"
	"github.com/openshift/hypershift/control-plane-operator/controllers/hostedcontrolplane/manifests"
	"github.com/openshift/hypershift/control-plane-operator/controllers/hostedcontrolplane/pki"
	"github.com/openshift/hypershift/control-plane-operator/controllers/hostedcontrolplane/util"
)

var (
	volumeMounts = util.PodVolumeMounts{
		oasContainerMain().Name: {
			oasVolumeWorkLogs().Name:           "/var/log/openshift-apiserver",
			oasVolumeConfig().Name:             "/etc/kubernetes/config",
			oasVolumeAuditConfig().Name:        "/etc/kubernetes/audit-config",
			oasVolumeAggregatorClientCA().Name: "/etc/kubernetes/certs/aggregator-client-ca",
			oasVolumeEtcdClientCA().Name:       "/etc/kubernetes/certs/etcd-client-ca",
			oasVolumeServingCA().Name:          "/etc/kubernetes/certs/serving-ca",
			oasVolumeKubeconfig().Name:         "/etc/kubernetes/secrets/svc-kubeconfig",
			oasVolumeServingCert().Name:        "/etc/kubernetes/certs/serving",
			oasVolumeEtcdClientCert().Name:     "/etc/kubernetes/certs/etcd-client",
		},
		oasKonnectivityProxyContainer().Name: {
			oasVolumeConfig().Name:                "/etc/kubernetes/config",
			oasVolumeKonnectivityProxyCert().Name: "/etc/konnectivity-proxy-tls",
		},
	}
	openShiftAPIServerLabels = map[string]string{
		"app": "openshift-apiserver",
	}
)

func ReconcileDeployment(deployment *appsv1.Deployment, ownerRef config.OwnerRef, config config.DeploymentConfig, image string, haproxyImage string, etcdURL string) error {
	ownerRef.ApplyTo(deployment)

	maxUnavailable := intstr.FromInt(1)
	maxSurge := intstr.FromInt(3)

	deployment.Spec.Strategy = appsv1.DeploymentStrategy{
		Type: appsv1.RollingUpdateDeploymentStrategyType,
		RollingUpdate: &appsv1.RollingUpdateDeployment{
			MaxUnavailable: &maxUnavailable,
			MaxSurge:       &maxSurge,
		},
	}
	deployment.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: openShiftAPIServerLabels,
	}
	deployment.Spec.Template.ObjectMeta.Labels = openShiftAPIServerLabels
	etcdUrlData, err := url.Parse(etcdURL)
	if err != nil {
		return fmt.Errorf("failed to parse etcd url: %w", err)
	}
	deployment.Spec.Template.Spec = corev1.PodSpec{
		AutomountServiceAccountToken: pointer.BoolPtr(false),
		Containers: []corev1.Container{
			util.BuildContainer(oasContainerMain(), buildOASContainerMain(image, strings.Split(etcdUrlData.Host,":")[0])),
			util.BuildContainer(oasKonnectivityProxyContainer(), buildOASKonnectivityProxyContainer(haproxyImage)),
		},
		Volumes: []corev1.Volume{
			util.BuildVolume(oasVolumeWorkLogs(), buildOASVolumeWorkLogs),
			util.BuildVolume(oasVolumeConfig(), buildOASVolumeConfig),
			util.BuildVolume(oasVolumeAuditConfig(), buildOASVolumeAuditConfig),
			util.BuildVolume(oasVolumeAggregatorClientCA(), buildOASVolumeAggregatorClientCA),
			util.BuildVolume(oasVolumeEtcdClientCA(), buildOASVolumeEtcdClientCA),
			util.BuildVolume(oasVolumeServingCA(), buildOASVolumeServingCA),
			util.BuildVolume(oasVolumeKubeconfig(), buildOASVolumeKubeconfig),
			util.BuildVolume(oasVolumeServingCert(), buildOASVolumeServingCert),
			util.BuildVolume(oasVolumeEtcdClientCert(), buildOASVolumeEtcdClientCert),
			util.BuildVolume(oasVolumeKonnectivityProxyCert(), buildOASVolumeKonnectivityProxyCert),
		},
	}

	config.ApplyTo(deployment)

	return nil
}

func oasContainerMain() *corev1.Container {
	return &corev1.Container{
		Name: "openshift-apiserver",
	}
}

func oasKonnectivityProxyContainer() *corev1.Container {
	return &corev1.Container{
		Name: "oas-konnectivity-proxy",
	}
}

func buildOASKonnectivityProxyContainer(routerImage string) func(c *corev1.Container) {
	return func(c *corev1.Container) {
		cpath := func(volume, file string) string {
			return path.Join(volumeMounts.Path(c.Name, volume), file)
		}
		c.Image = routerImage
		c.Command = []string{
			"/bin/bash",
			"-c",
		}
		c.Args = []string{
			fmt.Sprintf("cat %s %s > %s; haproxy -f %s",cpath(oasVolumeKonnectivityProxyCert().Name, "tls.crt"), cpath(oasVolumeKonnectivityProxyCert().Name, "tls.key"), haproxyCombinedPemLocation, cpath(oasVolumeConfig().Name, oasKonnectivityProxyConfigKey)),
		}
		c.VolumeMounts = volumeMounts.ContainerMounts(c.Name)
	}
}

func buildOASContainerMain(image string, etcdHostname string) func(c *corev1.Container) {
	return func(c *corev1.Container) {
		cpath := func(volume, file string) string {
			return path.Join(volumeMounts.Path(c.Name, volume), file)
		}
		c.Image = image
		c.Args = []string{
			"start",
			fmt.Sprintf("--config=%s", cpath(oasVolumeConfig().Name, openshiftAPIServerConfigKey)),
			fmt.Sprintf("--authorization-kubeconfig=%s", cpath(oasVolumeKubeconfig().Name, kas.KubeconfigKey)),
			fmt.Sprintf("--authentication-kubeconfig=%s", cpath(oasVolumeKubeconfig().Name, kas.KubeconfigKey)),
			fmt.Sprintf("--requestheader-client-ca-file=%s", cpath(oasVolumeAggregatorClientCA().Name, pki.CASignerCertMapKey)),
			"--requestheader-allowed-names=kube-apiserver-proxy,system:kube-apiserver-proxy,system:openshift-aggregator",
			"--requestheader-username-headers=X-Remote-User",
			"--requestheader-group-headers=X-Remote-Group",
			"--requestheader-extra-headers-prefix=X-Remote-Extra-",
			"--client-ca-file=/etc/kubernetes/config/serving-ca.crt",
			fmt.Sprintf("--client-ca-file=%s", cpath(oasVolumeServingCA().Name, pki.CASignerCertMapKey)),
		}
		c.Env = []corev1.EnvVar{
			{
				Name:  "HTTP_PROXY",
				Value: fmt.Sprintf("http://127.0.0.1:%d", konnectivity.KonnectivityServerLocalPort),
			},
			{
				Name:  "HTTPS_PROXY",
				Value: fmt.Sprintf("https://127.0.0.1:%d", konnectivity.KonnectivityServerLocalPort),
			},
			{
				Name:  "NO_PROXY",
				Value: fmt.Sprintf("%s,%s", manifests.KubeAPIServerService("").Name,etcdHostname),
			},
		}
		c.VolumeMounts = volumeMounts.ContainerMounts(c.Name)
		c.WorkingDir = volumeMounts.Path(oasContainerMain().Name, oasVolumeWorkLogs().Name)
	}
}

func oasVolumeWorkLogs() *corev1.Volume {
	return &corev1.Volume{
		Name: "work-logs",
	}
}

func buildOASVolumeWorkLogs(v *corev1.Volume) {
	v.EmptyDir = &corev1.EmptyDirVolumeSource{}
}

func oasVolumeConfig() *corev1.Volume {
	return &corev1.Volume{
		Name: "config",
	}
}

func buildOASVolumeConfig(v *corev1.Volume) {
	v.ConfigMap = &corev1.ConfigMapVolumeSource{}
	v.ConfigMap.Name = manifests.OpenShiftAPIServerConfig("").Name
}

func oasVolumeAuditConfig() *corev1.Volume {
	return &corev1.Volume{
		Name: "audit-config",
	}
}

func buildOASVolumeAuditConfig(v *corev1.Volume) {
	v.ConfigMap = &corev1.ConfigMapVolumeSource{}
	v.ConfigMap.Name = manifests.OpenShiftAPIServerAuditConfig("").Name
}

func oasVolumeKubeconfig() *corev1.Volume {
	return &corev1.Volume{
		Name: "kubeconfig",
	}
}

func buildOASVolumeKubeconfig(v *corev1.Volume) {
	v.Secret = &corev1.SecretVolumeSource{}
	v.Secret.SecretName = manifests.KASServiceKubeconfigSecret("").Name
}

func oasVolumeAggregatorClientCA() *corev1.Volume {
	return &corev1.Volume{
		Name: "aggregator-client-ca",
	}
}

func buildOASVolumeAggregatorClientCA(v *corev1.Volume) {
	v.Secret = &corev1.SecretVolumeSource{}
	v.Secret.SecretName = manifests.RootCASecret("").Name
}

func oasVolumeEtcdClientCA() *corev1.Volume {
	return &corev1.Volume{
		Name: "etcd-client-ca",
	}
}

func buildOASVolumeEtcdClientCA(v *corev1.Volume) {
	v.Secret = &corev1.SecretVolumeSource{}
	v.Secret.SecretName = manifests.RootCASecret("").Name
}

func oasVolumeServingCA() *corev1.Volume {
	return &corev1.Volume{
		Name: "serving-ca",
	}
}

func buildOASVolumeServingCA(v *corev1.Volume) {
	v.Secret = &corev1.SecretVolumeSource{}
	v.Secret.SecretName = manifests.RootCASecret("").Name
}

func oasVolumeServingCert() *corev1.Volume {
	return &corev1.Volume{
		Name: "serving-cert",
	}
}

func buildOASVolumeServingCert(v *corev1.Volume) {
	v.Secret = &corev1.SecretVolumeSource{}
	v.Secret.SecretName = manifests.OpenShiftAPIServerCertSecret("").Name
}

func oasVolumeEtcdClientCert() *corev1.Volume {
	return &corev1.Volume{
		Name: "etcd-client-cert",
	}
}

func buildOASVolumeEtcdClientCert(v *corev1.Volume) {
	v.Secret = &corev1.SecretVolumeSource{}
	v.Secret.SecretName = manifests.EtcdClientSecret("").Name
}

func oasVolumeKonnectivityProxyCert() *corev1.Volume {
	return &corev1.Volume{
		Name: "oas-konnectivity-proxy-cert",
	}
}

func buildOASVolumeKonnectivityProxyCert(v *corev1.Volume) {
	v.Secret = &corev1.SecretVolumeSource{}
	v.Secret.SecretName = manifests.KonnectivityClientSecret("").Name
}
