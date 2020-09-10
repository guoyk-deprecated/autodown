package main

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"log"
	"os"
	"time"
)

const (
	AnnotationLease = "net.guoyk.autodown/lease"
)

func exit(err *error) {
	if *err != nil {
		log.Println("exited with error:", (*err).Error())
		os.Exit(1)
	} else {
		log.Println("exited")
	}
}

func main() {
	log.SetOutput(os.Stdout)
	log.SetPrefix("[autodown] ")

	var err error
	defer exit(&err)

	var cfg *rest.Config
	if cfg, err = rest.InClusterConfig(); err != nil {
		return
	}
	var client *kubernetes.Clientset
	if client, err = kubernetes.NewForConfig(cfg); err != nil {
		return
	}

	var nss *corev1.NamespaceList
	if nss, err = client.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{}); err != nil {
		return
	}

	for _, ns := range nss.Items {
		log.Println("-- namespace:", ns.Name)
		// deployments
		var dps *appsv1.DeploymentList
		if dps, err = client.AppsV1().Deployments(ns.Name).List(context.Background(), metav1.ListOptions{}); err != nil {
			return
		}
		for _, dp := range dps.Items {
			if dp.Annotations == nil {
				log.Printf("deployment: %s, no annotations", dp.Name)
				continue
			}
			var leaseStr string
			if leaseStr = dp.Annotations[AnnotationLease]; leaseStr == "" {
				log.Printf("deployment: %s, no lease", dp.Name)
				continue
			}
			var lease time.Duration
			if lease, err = time.ParseDuration(leaseStr); err != nil {
				log.Printf("deployment: %s, failed to parse lease duration '%s': %s", dp.Name, leaseStr, err.Error())
				err = nil
				continue
			}
			var updatedAt time.Time
			for _, cond := range dp.Status.Conditions {
				var condTime = cond.LastTransitionTime.Time
				if !condTime.IsZero() && (updatedAt.IsZero() || condTime.After(updatedAt)) {
					updatedAt = condTime
				}
			}
			if updatedAt.IsZero() {
				log.Printf("deployment: %s, failed to determine updated at", dp.Name)
				continue
			}
			if time.Since(updatedAt) < lease {
				log.Printf("deployment: %s, nothing to do", dp.Name)
				continue
			}
			if _, err = client.AppsV1().Deployments(ns.Name).UpdateScale(context.Background(), dp.Name, &autoscalingv1.Scale{Spec: autoscalingv1.ScaleSpec{Replicas: 0}}, metav1.UpdateOptions{}); err != nil {
				return
			}
			log.Printf("deployment: %s, scaled to 0", dp.Name)
		}
		// statefulsets
		var sts *appsv1.StatefulSetList
		if sts, err = client.AppsV1().StatefulSets(ns.Name).List(context.Background(), metav1.ListOptions{}); err != nil {
			return
		}
		for _, st := range sts.Items {
			if st.Annotations == nil {
				log.Printf("statefulset: %s, no annotations", st.Name)
				continue
			}
			var leaseStr string
			if leaseStr = st.Annotations[AnnotationLease]; leaseStr == "" {
				log.Printf("statefulset: %s, no lease", st.Name)
				continue
			}
			var lease time.Duration
			if lease, err = time.ParseDuration(leaseStr); err != nil {
				log.Printf("statefulset: %s, failed to parse lease duration '%s': %s", st.Name, leaseStr, err.Error())
				err = nil
				continue
			}
			var updatedAt time.Time
			for _, cond := range st.Status.Conditions {
				var condTime = cond.LastTransitionTime.Time
				if !condTime.IsZero() && (updatedAt.IsZero() || condTime.After(updatedAt)) {
					updatedAt = condTime
				}
			}
			if updatedAt.IsZero() {
				log.Printf("statefulset: %s, failed to determine updated at", st.Name)
				continue
			}
			if time.Since(updatedAt) < lease {
				log.Printf("statefulset: %s, nothing to do", st.Name)
				continue
			}
			if _, err = client.AppsV1().StatefulSets(ns.Name).UpdateScale(context.Background(), st.Name, &autoscalingv1.Scale{Spec: autoscalingv1.ScaleSpec{Replicas: 0}}, metav1.UpdateOptions{}); err != nil {
				return
			}
			log.Printf("statefulset: %s, scaled to 0", st.Name)
		}
	}
}
