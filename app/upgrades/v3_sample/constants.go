package v3_sample

const (
	// UpgradeName is the shared upgrade plan name for mainnet
	UpgradeName = "v3.0.0"
	// DevnetUpgradeHeight defines the devnet block height on which the upgrade will take place
	DevnetUpgradeHeight = 999_999_999
	// UpgradeInfo defines the binaries that will be used for the upgrade
	UpgradeInfo = `'{"binaries":{"darwin/amd64":"https://github.com/europa/europa/releases/download/v3.0.0/europa_3.0.0_Darwin_arm64.tar.gz","darwin/x86_64":"https://github.com/europa/europa/releases/download/v3.0.0/europa_3.0.0_Darwin_x86_64.tar.gz","linux/arm64":"https://github.com/europa/europa/releases/download/v3.0.0/europa_3.0.0_Linux_arm64.tar.gz","linux/amd64":"https://github.com/europa/europa/releases/download/v3.0.0/europa_3.0.0_Linux_amd64.tar.gz","windows/x86_64":"https://github.com/europa/europa/releases/download/v3.0.0/europa_3.0.0_Windows_x86_64.zip"}}'`
)
