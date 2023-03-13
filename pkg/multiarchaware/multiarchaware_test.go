package multiarchaware

import (
	"context"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"testing"
)

func getOnlyAMD64Pod() *v1.Pod {
	return &v1.Pod{
		Spec: v1.PodSpec{
			Volumes: nil,
			InitContainers: []v1.Container{
				{
					Name:  "init",
					Image: "quay.io/openshift/origin-cli:latest",
				},
			},
			Containers: []v1.Container{
				{
					Name:  "container",
					Image: "quay.io/openshifttest/hello-openshift:1.2.0",
				},
			},
		},
	}
}

func getOnlyARM64Pod() *v1.Pod {
	return &v1.Pod{
		Spec: v1.PodSpec{
			Volumes: nil,
			InitContainers: []v1.Container{
				{
					Name:  "init",
					Image: "quay.io/openshift/origin-cli:latest",
				},
			},
			Containers: []v1.Container{
				{
					Name:  "container",
					Image: "quay.io/openshifttest/hello-openshift:1.2.0",
				},
			},
		},
	}
}

func getMultiarchPod() *v1.Pod {
	return &v1.Pod{
		Spec: v1.PodSpec{
			Volumes: nil,
			InitContainers: []v1.Container{
				{
					Name:  "init",
					Image: "quay.io/openshifttest/hello-openshift:1.2.0",
				},
			},
			Containers: []v1.Container{
				{
					Name:  "container",
					Image: "quay.io/openshifttest/hello-openshift:1.2.0",
				},
			},
		},
	}
}

func getNode() *v1.Node {
	return &v1.Node{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "node",
			Labels: map[string]string{
				"kubernetes.io/arch": "arm64",
			},
			Annotations:     nil,
			OwnerReferences: nil,
			Finalizers:      nil,
			ManagedFields:   nil,
		},
		Spec:   v1.NodeSpec{},
		Status: v1.NodeStatus{},
	}
}

func TestMultiArchPod(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	pod := getMultiarchPod()
	node := getNode()
	plugin := MultiArchAware{fh: nil}
	nodeInfo := &framework.NodeInfo{
		Pods:                         nil,
		PodsWithAffinity:             nil,
		PodsWithRequiredAntiAffinity: nil,
		UsedPorts:                    nil,
		Requested:                    nil,
		NonZeroRequested:             nil,
		Allocatable:                  nil,
		ImageStates:                  nil,
		PVCRefCounts:                 nil,
		Generation:                   0,
	}
	nodeInfo.SetNode(node)
	ret := plugin.Filter(context.TODO(), nil, pod, nodeInfo)
	if ret.Code() != framework.Success {
		t.Errorf("expected schedulable, got %v", ret.Code())
	}
}

func TestAMD64OnlyPodOnARM64Node(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	pod := getOnlyAMD64Pod()
	node := getNode()
	plugin := MultiArchAware{fh: nil}
	nodeInfo := &framework.NodeInfo{
		Pods:                         nil,
		PodsWithAffinity:             nil,
		PodsWithRequiredAntiAffinity: nil,
		UsedPorts:                    nil,
		Requested:                    nil,
		NonZeroRequested:             nil,
		Allocatable:                  nil,
		ImageStates:                  nil,
		PVCRefCounts:                 nil,
		Generation:                   0,
	}
	nodeInfo.SetNode(node)
	ret := plugin.Filter(context.TODO(), nil, pod, nodeInfo)
	if ret.Code() != framework.Unschedulable {
		t.Errorf("expected unschedulable, got %v", ret.Code())
	}
}
