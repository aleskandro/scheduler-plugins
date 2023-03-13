package multiarchaware

import (
	"context"
	"fmt"
	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/image"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/types"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

type MultiArchAware struct {
	fh framework.Handle
}

const Name = "MultiArchAware"

type void struct{}

var voidVal void
var _ framework.FilterPlugin = &MultiArchAware{}

// Name returns name of the plugin. It is used in logs, etc.
func (pl *MultiArchAware) Name() string {
	return Name
}

// New initializes a new plugin and returns it.
func New(_ runtime.Object, fh framework.Handle) (framework.Plugin, error) {
	klog.InfoS("Initializing Multi-arch aware scheduler plugin")
	pl := MultiArchAware{
		fh: fh,
	}
	return &pl, nil
}

// Filter invoked at the filter extension point.
func (pl *MultiArchAware) Filter(ctx context.Context, state *framework.CycleState,
	pod *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	// Build a set of all the images used by the pod
	imageNamesSet := map[string]void{}
	for _, container := range append(pod.Spec.Containers, pod.Spec.InitContainers...) {
		imageNamesSet[fmt.Sprintf("//%s", container.Image)] = voidVal
	}
	klog.Infof("[multiarch-aware] Images list: %v", imageNamesSet)
	nodeArchitecture := nodeInfo.Node().ObjectMeta.Labels["kubernetes.io/arch"]
	if nodeArchitecture == "" {
		klog.InfoS("[multiarch-aware] Node architecture not found, ignoring the multi-arch aware scheduling filter predicates." +
			" This may lead to scheduling pods on nodes with incompatible architectures.")
		return framework.NewStatus(framework.Success, "Node architecture not found. Ignoring multi-arch aware filtering plugin")
	}

	// https://github.com/containers/skopeo/blob/v1.11.1/cmd/skopeo/inspect.go#L72
	// Iterate over the images and check their architectures
	var (
		ref types.ImageReference
		err error
		src types.ImageSource
	)

	for imageName := range imageNamesSet {
		klog.Infof("[multiarch-aware] Checking image: %s", imageName)
		// Check if the imageName is a multi-arch imageName
		ref, err = docker.ParseReference(imageName)
		if err != nil {
			// Ignore errors due to invalid images at this stage
			klog.Infof("Error parsing imageName reference: %v", err)
			continue
		}
		src, err = ref.NewImageSource(ctx, &types.SystemContext{})
		if err != nil {
			// Ignore errors due to invalid images at this stage
			klog.Infof("Error creating imageName source: %v", err)
			continue
		}
		// defer src.Close()
		rawManifest, _, err := src.GetManifest(ctx, nil)
		if err != nil {
			// Ignore errors due to invalid images at this stage
			klog.Infof("Error getting imageName manifest: %v", err)
			continue
		}
		if manifest.MIMETypeIsMultiImage(manifest.GuessMIMEType(rawManifest)) {
			klog.Infof("[multiarch-aware] %s is a ManifestList image... Checking existence "+
				"of a manifest for the node %s's architecture (%s)",
				imageName, nodeInfo.Node().Name, nodeArchitecture)
			// The imageName is a Manifest List
			index, err := manifest.OCI1IndexFromManifest(rawManifest)
			if err != nil {
				// Ignore errors due to invalid images at this stage
				klog.Infof("Error parsing imageName manifest: ", err)
				continue
			}
			notFound := true
			for _, m := range index.Manifests {
				if m.Platform.Architecture == nodeArchitecture {
					// The sets intersect, the imageName can run on this architecture
					klog.Infof("[multiarch-aware] %s has a manifest for the node %s's architecture (%s)",
						imageName, nodeInfo.Node().Name, nodeArchitecture)
					notFound = false
				}
			}
			if notFound {
				klog.Infof("[multiarch-aware] %s does not have a manifest for the node %s's architecture (%s)",
					imageName, nodeInfo.Node().Name, nodeArchitecture)

				// The imageName cannot run on this node
				return framework.NewStatus(framework.Unschedulable, "The imageName does not support the architecture of this node")
			}
		} else {
			// The imageName is not a Manifest List
			klog.Infof("[multiarch-aware] %s is not a ManifestList image... Checking the architecture of the image",
				imageName)

			// SystemContext should be filled with the proper information from the node (credentials included).
			sys := &types.SystemContext{}
			parsedImage, err := image.FromUnparsedImage(ctx, sys, image.UnparsedInstance(src, nil))
			if err != nil {
				// Ignore errors due to invalid images at this stage
				klog.Infof("Error parsing the manifest: %v", err)
				continue
			}
			config, err := parsedImage.OCIConfig(ctx)
			if err != nil {
				// Ignore errors due to invalid images at this stage
				klog.Infof("Error parsing the OCI config: %v", err)
				continue
			}
			if config.Architecture != nodeArchitecture {
				// The imageName cannot run on this node
				klog.Infof("[multiarch-aware] %s is build for architecture %s that is not compatible with the node %s's architecture (%s)",
					imageName, config.Architecture, nodeInfo.Node().Name, nodeArchitecture)
				return framework.NewStatus(framework.Unschedulable, "The imageName does not support the architecture of this node")
			}
		}
	}
	klog.Infof("[multiarch-aware] All images are compatible with the node's architecture")
	return framework.NewStatus(framework.Success)
}
