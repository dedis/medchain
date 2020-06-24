
# MedChain Simulation

After you know that a new service and protocol work at all (unit testing and integration testing have passed) then you might want to know how that software will behave in larger networks. This is the job of the simulation system.

References:
* The README.md: https://github.com/dedis/onet/blob/master/simul/README.md
* The interface you need to implement: https://godoc.org/github.com/dedis/onet#Simulation

Simulations in onet are a powerful way of making sure your code is well behaving
also in bigger settings, including on different servers, and of course to write
simulations used in research papers to make pretty graphs.
In order to write a simulation, you must make a struct that implements the [onet.Simulation](https://godoc.org/github.com/dedis/onet#Simulation) interface.

You will need to implement the `Setup` method to return the
`*onet.SimulationConfig` instance and to create a roster and a tree. The `Setup`
method is run at the beginning of the simulation, on your computer. It prepares
all the necessary structures and can also copy needed files for the actual simulation
run.

The `Node` method is run before the actual simulation is started and is called
once for every node. The simulation framework makes sure that all nodes have
finished their `Node` method before the `Run` is called.

The `Run` method is only called on the root node, which is the first node of
the Roster.

## Running your simulation

The simulation for MedChain is defined in `service.go`. It simulates a client
talking to a service (files `service.go` and `service.toml`). File `service.toml` defines the simulation and its setup (for tests on localhost).

You can build the simulator executable with `go build`. If you try to run it
with no options (`./simulator`), it asks for a simulation to run. You must give
it one or more toml files on the commandline, i.e., 

```bash
$ ./simulation service.toml
```

Simulation results and time measurements are written to `test_data` folder that is created once you run the simulations. You can use the provided python script to produce the plots of results:

```bash
$ python plot_simulation.py
```

The plots will be saved in `test_data` folder.

## Running simulations with "go test"

The `simul_test.go` file shows that simulations can be launched from within
standard Go tests. This would make it possible to have different tests that use
different toml files in order to test different sizes of networks, etc.


## Directory overview

Below is the description of code and files avaliable in this directory:

- `service.go`: Implementation of MedChain service simulation 
- `service.toml`: toml file for service simulations
- `simul.go`: Simulation code using Go tests 
- `simul_test.go`: Simulation code using Go tests
- `utils.go`: Simulation utility code