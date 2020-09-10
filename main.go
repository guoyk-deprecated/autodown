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
	"strings"
	"time"
)

const (
	AnnotationLease    = "net.guoyk.autodown/lease"
	AnnotationDisabled = "net.guoyk.autodown/disabled"

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

func buildLoggerWhitespaces(l int) string {
	if l < 12 {
		return strings.Repeat(" ", 12-l)
	} else if l < 24 {
		return strings.Repeat(" ", 24-l)
	} else if l < 36 {
		return strings.Repeat(" ", 36-l)
	} else if l < 48 {
		return strings.Repeat(" ", 48-l)
	} else {
		return ""
	}
}

func buildLogger(dp string) func(s string) {
	sb := &strings.Builder{}
	sb.WriteString("└ deployment: [")
	sb.WriteString(dp)
	sb.WriteString("] ")
	sb.WriteString(buildLoggerWhitespaces(len(dp)))
	h := sb.String()
	return func(s string) {
		log.Println(h + s)
	}
}

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ltime | log.Lmsgprefix)
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
		log.Printf("namespace: [%s]", ns.Name)

		var dpList *appsv1.DeploymentList
		if dpList, err = client.AppsV1().Deployments(ns.Name).List(context.Background(), metav1.ListOptions{}); err != nil {
			return
		}

		for _, dp := range dpList.Items {
			dpLog := buildLogger(dp.Name)
			// check annotations exists
			if dp.Annotations == nil {
				continue
			}
			// get lease annotation
			var leaseStr string
			if leaseStr = dp.Annotations[AnnotationLease]; leaseStr == "" {
				continue
			}
			// get disabled annotation
			if disabled, _ := strconv.ParseBool(dp.Annotations[AnnotationDisabled]); disabled {
				dpLog("disabled, skipping")
				continue
			}
			// parse lease annotation
			var lease time.Duration
			if lease, err = time.ParseDuration(leaseStr); err != nil {
				dpLog("failed to parse lease, skipping")
				err = nil
				continue
			}
			// check replicas
			if dp.Status.Replicas == 0 {
				dpLog("already scaled to 0, skipping")
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
				dpLog("failed to determine last update time, skipping")
				continue
			}
			if time.Since(updateTime) < lease {
				dpLog("lease not expired, skipping")
				continue
			}
			// scale to 0
			if !optDryRun {
				if _, err = client.AppsV1().Deployments(dp.Namespace).Patch(context.Background(), dp.Name, types.StrategicMergePatchType, []byte(patchReplicasZero), metav1.PatchOptions{}); err != nil {
					return
				}
			}
			dpLog("scaled to 0")
		}
	}
}
