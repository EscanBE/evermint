package upgrade

import (
	"github.com/europa/europa/v12/constants"
)

// The constants used in the upgrade tests are defined here
const (
	// the defaultChainID used for testing
	defaultChainID = constants.TestnetFullChainId

	// LocalVersionTag defines the docker image ImageTag when building locally
	LocalVersionTag = "latest"

	// dockerRepo is the docker hub repository that contains our chain app's images pulled during tests
	dockerRepo = "europa/europa"

	// upgradesPath is the relative path from this folder to the app/upgrades folder
	upgradesPath = "../../../app/upgrades"

	// versionSeparator is used to separate versions in the INITIAL_VERSION and TARGET_VERSION
	// environment vars
	versionSeparator = "/"
)
