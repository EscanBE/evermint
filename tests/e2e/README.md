# End-to-End Testing Suite

The End-to-End (E2E) testing suite provides an environment
for running end-to-end tests on our chain.
It is used for testing chain upgrades,
as it allows for initializing multiple Europa chains with different versions.

- [End-to-End Testing Suite](#end-to-end-testing-suite)
    - [Quick Start](#quick-start)
    - [Upgrade Process](#upgrade-process)
    - [Test Suite Structure](#test-suite-structure)
        - [`e2e` Package](#e2e-package)
        - [`upgrade` Package](#upgrade-package)
        - [Version retrieve](#version-retrieve)
        - [Testing Results](#testing-results)
    - [Running multiple upgrades](#running-multiple-upgrades)

### Quick Start

To run the e2e tests, execute:

```shell
make test-e2e
```

This command runs an upgrade test (upgrading a node from an old version to a newer one),
as well as query and transactions operations against a node with the latest changes.

This logic utilizes parameters that can be set manually(if necessary):

```shell
# flag to skip containers cleanup after upgrade
# should be set true with make test-e2e command if you need access to the node
# after upgrading
E2E_SKIP_CLEANUP := false

# version(s) of initial node(s) that will be upgraded, tag e.g. 'v12.1.0'
# to use multiple upgrades separate the versions with a forward slash, e.g.
# 'v12.0.1/v12.0.2'
INITIAL_VERSION

# version of upgraded node that will replace the initial node, tag e.g.
# 'v13.0.0'
TARGET_VERSION

# mount point for the upgraded node container, to mount new node version to
# previous node state folder. By default this is './build/.euscd:/root/.euscd'
# More info at https://docs.docker.com/engine/reference/builder/#volume
MOUNT_PATH

# '--chain-id' our chain's cli parameter, used to start nodes with a specific
# chain-id and submit proposals
# By default this is 'europa_80808-1'
CHAIN_ID
```

To test an upgrade to explicit target version
and continue to run the upgraded node, use:

```shell
make test-e2e E2E_SKIP_CLEANUP=true INITIAL_VERSION=<tag> TARGET_VERSION=<tag>
```

### Upgrade Process

Testing a chain upgrade is a multi-step process:

1. Build a docker image for the target version of our chain
(local repo by default, if no explicit `TARGET_VERSION` provided as argument)
(e.g. `v12.0.0`)
2. Run tests
3. The e2e test will first run an `INITIAL_VERSION` node container.
4. The node will submit, deposit and vote for an upgrade proposal
for upgrading to the `TARGET_VERSION`.
5. After block `50` is reached,
the test suite exports `/.euscd` folder from the docker container
to the local `build/` folder and then purges the container.
6. Suite will mount the node with `TARGET_VERSION`
to the local `build/` dir and start the node.
The node will get upgrade information from `upgrade-info.json`
and will execute the upgrade.

## Test Suite Structure

### `e2e` Package

The `e2e` package defines an integration testing suite
used for full end-to-end testing functionality.
This package is decoupled from depending on the Europa codebase.
It initializes the chains for testing via Docker.  
As a result, the test suite may provide the
desired version to Docker containers during the initialization.
This design allows for the opportunity of testing chain upgrades
by providing an older version to the container,
performing the chain upgrade,
and running the latest test suite.  
Here's an overview of the files:

* `e2e_suite_test.go`: defines the testing suite
and contains the core bootstrapping logic
that creates a testing environment via Docker containers.
A testing network is created dynamically with 2 test validators.

* `e2e_test.go`: contains the actual end-to-end integration tests
that utilize the testing suite.

* `e2e_utils_test.go`: contains suite upgrade params loading logic.

### `upgrade` Package

The `e2e` package defines an upgrade `Manager` abstraction.
Suite will utilize `Manager`'s functions
to run different versions of containers running our chain,
propose, vote, delegate and query nodes.

* `manager.go`: defines core manager logic for running containers,
export state and create networks.

* `govexec.go`: defines `gov-specific` exec commands to submit/delegate/vote
through nodes `gov` module.

* `node.go`: defines `Node` structure
responsible for setting node container parameters before run.

### Version retrieve

If `INITIAL_VERSION` is provided as an argument,
node container(s) with the corresponding version(s)
will be pulled from [DockerHub](https://hub.docker.com/r/europa/europa/tags).
If it is not specified,
the test suite retrieves the second-to-last upgrade version
from the local codebase (in the `europa/app/upgrades` folder)
according to [Semantic Versioning](https://semver.org/).

If `TARGET_VERSION` is specified,
the corresponding container will also be pulled from DockerHub.
When not specified, the test suite will retrieve the latest upgrade version
from `europa/app/upgrades`.

### Testing Results

The `make test-e2e` script will output the test results
for each testing file.
In case of a successful upgrade,
the script will print the following output (example):

```log
ok  	github.com/europa/europa/v12/tests/e2e	174.137s.
```

If the target node version fails to start,
the logs from the docker container will be printed:

```log
Error:  Received unexpected error:
        can't start node, container exit code: 2

        [error stream]:

        7:03AM INF Unlocking keyring
        7:03AM INF starting ABCI with Tendermint
        panic: invalid minimum gas prices: invalid decimal coin expression: 0...

        goroutine 1 [running]:
        github.com/cosmos/cosmos-sdk/baseapp.SetMinGasPrices({0xc0013563e7?, ...
            github.com/cosmos/cosmos-sdk@v0.46.16/baseapp/options.go:29 +0xd9
        main.appCreator.newApp({{{0x3399b40, 0xc000ec1db8}, {0x33ac0f8, 0xc00...
            github.com/europa/europa/v12/cmd/euscd/root.go:243 +0x2ca

        [output stream]:

Test:     TestIntegrationTestSuite/TestUpgrade
Messages: can't mount and run upgraded node container
```

To get all containers run:

```shell
# list containers
docker ps -a
```

Container names will be listed as follows:

```log
CONTAINER ID   IMAGE
9307f5485323   europa:local    <-- upgraded node
f41c97d6ca21   europa:v12.0.0   <-- initial node
```

To access containers logs directly, run:

```shell
docker logs <container-id>
```

To interact with the upgraded node
pass `SKIP_CLEANUP=true` to the make command
and enter the container after the upgrade has finished:

```shell
docker exec -it <container-id> bash
```

If the cleanup was skipped
the upgraded node container should be removed manually:

```shell
docker kill <container-id>
docker rm <container-id>
```

## Running multiple upgrades

In order to run multiple upgrades,
just combine the versions leading up to the last upgrade
with a forward slash
and pass them as the `INITIAL_VERSION`.

```bash
make test-e2e INITIAL_VERSION=v10.0.1/v11.0.0-rc1 TARGET_VERSION=v11.0.0-rc3
```
