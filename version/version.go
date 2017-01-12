package version

var (
	// Version is the application version and should be specified at compile
	// time with, e.g., -ldflags "-X version.Version=x.y.z".
	Version string
	// ImageRepo is the container image repository and should be specified at
	// compile time with, e.g., "-X version.ImageRepo=gcr.io/puppet-panda-dev".
	ImageRepo string
	// ResourceAPIVersion is the API version for ThirdPartyResources.
	ResourceAPIVersion = "v1alpha1"
)
