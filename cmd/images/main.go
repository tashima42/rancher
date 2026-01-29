package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/rancher/rancher/cmd/images/appco"
	"github.com/rancher/rancher/cmd/images/utilities"
	img "github.com/rancher/rancher/pkg/image"
)

func main() {
	var (
		chartsPath           string
		ociChartsPath        string
		ociChartsRepoPrefix  string
		rancherTag           string
		imgs                 string
		enableAppcoArtifacts bool
	)

	flag.StringVar(&chartsPath, "charts-path", "", "path to the charts repository (required)")
	flag.StringVar(&ociChartsPath, "oci-charts-path", "", "path to the oci charts repository")
	flag.StringVar(&ociChartsRepoPrefix, "oci-charts-repo-prefix", "rancher/charts", "prefix of the oci charts repository")
	flag.StringVar(&rancherTag, "tag", "", "rancher target tag (required)")
	flag.StringVar(&imgs, "extra-images", "", "list of extra images")
	flag.BoolVar(&enableAppcoArtifacts, "enable-appco-artifacts", false, "enable appco artifacts read from oci charts directory")

	flag.Parse()

	if chartsPath == "" {
		fmt.Fprintln(os.Stderr, "error: required charts-path flag is empty")
		flag.Usage()
		os.Exit(1)
	}

	if rancherTag == "" {
		fmt.Fprintln(os.Stderr, "error: required tag flag is empty")
		flag.Usage()
		os.Exit(1)
	}

	extraImages := strings.Split(imgs, ",")

	if err := run(chartsPath, extraImages, ociChartsPath, ociChartsRepoPrefix, rancherTag, enableAppcoArtifacts); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func run(chartsPath string, imagesFromArgs []string, ociChartsPath string, ociRepositoryURL string, rancherTag string, enableAppcoArtifacts bool) error {
	targetsAndSources, err := utilities.GatherTargetArtifactsAndSources(chartsPath, ociChartsPath, imagesFromArgs, ociRepositoryURL, rancherTag)
	if err != nil {
		return err
	}

	// Append AppCo artifacts if enabled
	if enableAppcoArtifacts {
		appcoArtifacts, err := appco.CollectArtifacts()
		if err != nil {
			return err
		}

		// add to rancher-images.txt
		targetsAndSources.TargetLinuxArtifacts = append(targetsAndSources.TargetLinuxArtifacts, appcoArtifacts...)

		// add to rancher-images-sources.txt
		// Source is "appco" for all AppCo artifacts
		for _, artifact := range appcoArtifacts {
			targetsAndSources.TargetLinuxArtifactsAndSources = addSourceToImage(
				targetsAndSources.TargetLinuxArtifactsAndSources,
				artifact,
				"appco",
			)
		}
	}

	// create rancher-image-origins.txt. Will fail if /pkg/image/origins.go
	// does not provide a mapping for each image.
	err = img.GenerateImageOrigins(targetsAndSources.LinuxImagesFromArgs, targetsAndSources.TargetLinuxArtifacts, targetsAndSources.TargetWindowsArtifacts)
	if err != nil {
		return err
	}

	type imageTextLists struct {
		images           []string
		imagesAndSources []string
	}
	for arch, imageLists := range map[string]imageTextLists{
		"linux":   {images: targetsAndSources.TargetLinuxArtifacts, imagesAndSources: targetsAndSources.TargetLinuxArtifactsAndSources},
		"windows": {images: targetsAndSources.TargetWindowsArtifacts, imagesAndSources: targetsAndSources.TargetWindowsArtifactsAndSources},
	} {
		err = utilities.ImagesText(arch, imageLists.images)
		if err != nil {
			return err
		}

		if err = utilities.ImagesAndSourcesText(arch, imageLists.imagesAndSources); err != nil {
			return err
		}
		err = utilities.MirrorScript(arch, imageLists.images)
		if err != nil {
			return err
		}

		err = utilities.SaveScript(arch, imageLists.images)
		if err != nil {
			return err
		}

		err = utilities.LoadScript(arch, imageLists.images)
		if err != nil {
			return err
		}
	}

	return nil
}

func addSourceToImage(
	imagesAndSources []string,
	image string,
	source string,
) []string {
	if image == "" || source == "" {
		return imagesAndSources
	}

	return append(
		imagesAndSources,
		fmt.Sprintf("%s %s", image, source),
	)
}
