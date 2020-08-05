package test

import (
	"context"

	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	conformanceImage         = "k8s.gcr.io/conformance:corev1.18.6"
	conformanceNamespaceName = "conformance"
	conformanceResourceName  = "conformance"
)

type Config struct {
	Config *rest.Config
}

type Test struct {
	k8sClient kubernetes.Interface
}

func New(config Config) (*Test, error) {
	k8sClient, err := kubernetes.NewForConfig(config.Config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	test := Test{
		k8sClient: k8sClient,
	}

	return &test, nil
}

func (t *Test) RunAWSConformance() error {
	return nil
}

func (t *Test) RunKubernetesConformance(ctx context.Context) error {
	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: conformanceNamespaceName,
		},
	}
	_, err := t.k8sClient.CoreV1().Namespaces().Create(ctx, &namespace, metav1.CreateOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	serviceAccount := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: conformanceResourceName,
		},
	}
	_, err = t.k8sClient.CoreV1().ServiceAccounts(namespace.Name).Create(ctx, &serviceAccount, metav1.CreateOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	role := rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: conformanceResourceName,
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"*"},
				APIGroups: []string{"*"},
				Resources: []string{"*"},
			},
			{
				Verbs: []string{"get"},
				NonResourceURLs: []string{
					"/metrics",
					"/logs",
					"/logs/*",
				},
			},
		},
	}
	_, err = t.k8sClient.RbacV1().ClusterRoles().Create(ctx, &role, metav1.CreateOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	roleBinding := rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: conformanceResourceName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      serviceAccount.Name,
				Namespace: namespace.Name,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     role.Name,
		},
	}
	_, err = t.k8sClient.RbacV1().ClusterRoleBindings().Create(ctx, &roleBinding, metav1.CreateOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: conformanceResourceName,
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name: "output-volume",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/tmp/results",
						},
					},
				},
			},

			Containers: []corev1.Container{
				{
					Name:  "conformance-container",
					Image: conformanceImage,
					Env: []corev1.EnvVar{
						{
							Name:  "E2E_FOCUS",
							Value: "\\[Conformance\\]",
						},
						{
							Name:  "E2E_SKIP",
							Value: "",
						},
						{
							Name:  "E2E_PROVIDER",
							Value: "skeleton",
						},
						{
							Name:  "E2E_PARALLEL",
							Value: "false",
						},
						{
							Name:  "E2E_VERBOSITY",
							Value: "4",
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "output-volume",
							MountPath: "/tmp/results",
						},
					},
					ImagePullPolicy: "IfNotPresent",
				},
			},
			RestartPolicy:      "Never",
			ServiceAccountName: serviceAccount.Name,
		},
	}
	_, err = t.k8sClient.CoreV1().Pods(namespace.Name).Create(ctx, &pod, metav1.CreateOptions{})
	return nil
}

func (t *Test) RunCIS() error {
	return nil
}
