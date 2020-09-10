package main

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"log"
	"os"
	"strconv"
	"time"
)

const (
	AnnotationLease = "net.guoyk.autodown/lease"

	patchReplicasZero = `{"spec":{"replicas": 0}}`
)

var (
	optDryRun, _ = strconv.ParseBool(os.Getenv("AUTODOWN_DRY_RUN"))
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
	if optDryRun {
		log.SetPrefix("[autodown (dry)] ")
	} else {
		log.SetPrefix("[autodown] ")
	}

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

	var nsList *corev1.NamespaceList
	if nsList, err = client.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{}); err != nil {
		return
	}

	for _, ns := range nsList.Items {
		log.Println("namespace:", ns.Name)

		var dpList *appsv1.DeploymentList
		if dpList, err = client.AppsV1().Deployments(ns.Name).List(context.Background(), metav1.ListOptions{}); err != nil {
			return
		}

		for _, dp := range dpList.Items {
			// check annotations exists
			if dp.Annotations == nil {
				continue
			}
			// get lease annotation
			var leaseStr string
			if leaseStr = dp.Annotations[AnnotationLease]; leaseStr == "" {
				continue
			}
			// parse lease annotation
			var lease time.Duration
			if lease, err = time.ParseDuration(leaseStr); err != nil {
				log.Printf("  deployment: %s, failed to parse lease duration '%s': %s", dp.Name, leaseStr, err.Error())
				err = nil
				continue
			}
			// check replicas
			if dp.Status.Replicas == 0 {
				log.Printf("  deployment: %s, already scaled to 0", dp.Name)
				continue
			}
			// check update time
			var updateTime time.Time
			for _, cond := range dp.Status.Conditions {
				t := cond.LastUpdateTime.Time
				if !t.IsZero() && (updateTime.IsZero() || t.After(updateTime)) {
					updateTime = t
				}
			}
			if updateTime.IsZero() {
				log.Printf("  deployment: %s, failed to determine last update time", dp.Name)
				continue
			}
			if time.Since(updateTime) < lease {
				log.Printf("  deployment: %s, lease not yet expired", dp.Name)
				continue
			}
			// scale to 0
			if !optDryRun {
				if _, err = client.AppsV1().Deployments(dp.Namespace).Patch(context.Background(), dp.Name, types.StrategicMergePatchType, []byte(patchReplicasZero), metav1.PatchOptions{}); err != nil {
					return
				}
			}
			log.Printf("  deployment: %s, scaled to 0", dp.Name)
		}
	}
}
